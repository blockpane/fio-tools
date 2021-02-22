package bulk

import (
	"errors"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"strconv"
	"time"
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
	stat, err := F.Stat()
	if err != nil {
		return 0, err
	}
	if stat.Size() <= 1 {
		return 0, errors.New("empty file")
	}
	requests, e := slurp()
	if e != nil {
		return 0, err
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
