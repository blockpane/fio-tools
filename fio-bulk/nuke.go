package bulk

import (
	"bufio"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const csvHeader = `"request_id","payer","payer_fio","payee","payee_fio","address","amount","chain","token","memo","hash","url"` + "\n"

var (
	InFile  string
	OutFile string

	Acc *fio.Account
	Api *fio.API
	F   *os.File

	Quiet   bool
	Confirm bool
	Nuke    bool
	Verbose bool
	Query   bool
	SendFio float32
)

const (
	bundleReject = 1
	bundleRespond = 2
)

func NukeEmAll() (rejected int, err error) {
	var r fio.PendingFioRequestsResponse
	// rejecting again before committed? Maybe, this moves pretty quick. skip dups....
	dups := make(map[uint64]bool)

	retried := 0
	for {
		time.Sleep(50 * time.Millisecond)
		r, _, err = Api.GetPendingFioRequests(Acc.PubKey, 1000, 0)
		if len(r.Requests) == 0 {
			retried += 1
			// something's not right with the api responses, retry a few times. Sometimes getting an empty result
			// where there are pending requests!
			if retried > 3 {
				return
			}
			continue
		}
		for _, req := range r.Requests {
			if dups[req.FioRequestId] {
				continue
			}
			dups[req.FioRequestId] = true
			// closure to deref
			func(i uint64) {
				_, err = Api.SignPushActions(fio.NewRejectFndReq(Acc.Actor, strconv.Itoa(int(i))))
				if err != nil {
					log.Println(err)
				} else {
					log.Println("rejected request:", i)
					rejected += 1
				}
			}(req.FioRequestId)
		}
	}
}

func RejectRequests() (rejected int, err error) {
	requests := make([]string, 0)
	reader := bufio.NewReader(F)
	defer F.Close()
	var e error
	var l []byte
	for {
		l, _, e = reader.ReadLine()
		if e != nil {
			if e.Error() == "EOF" {
				break
			}
			return rejected, e
		}
		var id int
		id, e = strconv.Atoi(strings.TrimSpace(string(l)))
		if e != nil {
			log.Println("could not parse line:", l)
			continue
		}
		pending, e := IsPending(id)
		if e != nil {
			log.Println(err)
			continue
		}
		if pending {
			requests = append(requests, strconv.Itoa(id))
		} else {
			log.Println("have already responded to id", id, "skipping")
		}
	}
	if Verbose {
		fmt.Println(requests)
	}
	for _, id := range requests {
		resp := &eos.PushTransactionFullResp{}
		resp, err = Api.SignPushActions(fio.NewRejectFndReq(Acc.Actor, id))
		if err != nil {
			return
		}
		log.Printf("rejected %s with txid %s\n", id, resp.TransactionID)
		rejected += 1
	}
	return
}

type tinyResult struct {
	Id int `json:"id"`
}
