package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"net/url"
	"os"
	"sort"
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

func main() {
	log.SetFlags(log.LUTC | log.LstdFlags | log.Lshortfile)
	options()
	switch true {
	case Nuke:
		deleted, err := nukeEmAll()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Rejected %d requests\n", deleted)
		if r, _, _ := Api.GetPendingFioRequests(Acc.PubKey, 1000, 0); len(r.Requests) > 0 {
			log.Fatal("there are still pending requests, please try again.")
		}
		os.Exit(0)
	case Query:
		ok, wrote, err := dumpRequests()
		log.Printf("wrote %d records to %s\n", wrote, OutFile)
		if !ok {
			log.Println("WARNING: could not retrieve all records, the table row query may have timed out.")
		}
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	case Confirm:
		// handle positive responses here.
	}
	rejected, err := rejectRequests()
	log.Printf("rejected %d requests from %s\n", rejected, InFile)
	if err != nil {
		log.Fatal(err)
	}
}

func nukeEmAll() (rejected int, err error) {
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

func rejectRequests() (rejected int, err error) {
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
			continue
		}
		pending, e := isPending(id)
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

// isPending looks at the fioreqstss and determines if a response has already been sent for the request.
func isPending(id int) (bool, error) {
	i := strconv.Itoa(id)
	gtr, err := Api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqstss",
		Index:      "2",
		KeyType:    "i64",
		LowerBound: i,
		UpperBound: i,
		Limit:      1,
		JSON:       true,
	})
	if err != nil {
		return false, err
	}
	ids := make([]tinyResult, 0)
	err = json.Unmarshal(gtr.Rows, &ids)
	if err != nil {
		log.Println("request id:", id, err)
		return false, err
	}
	if ids != nil && len(ids) > 0 {
		return false, nil
	}
	return true, nil
}

// getAddrHashes returns a slice of i128 hashes for all the FIO addresses owned by the account,  which is a partial sha1 sum.
// in order to get all requests using a table lookup, it's necessary to know the address hash so we can query by a secondary key.
func getAddrHashes() ([]string, error) {
	n, _, err := Acc.GetNames(Api)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("did not find any FIO addresses for that key")
	}
	hashes := make([]string, n)
	for i := range Acc.Addresses {
		hashes[i] = fio.DomainNameHash(Acc.Addresses[i].FioAddress)
	}
	return hashes, nil
}

// onlyRequestId is a truncated version of the struct in the fioreqctxts table for extracting only the id.
type onlyRequestId struct {
	FioRequestId uint64 `json:"fio_request_id"`
}

// requestsFromTable uses a table query to (attempt to) bypass the limitation of the API endpoint where it will timeout when
// there are thousands of pending requests, it expects an i128 hash, and returns a slice of int64 representing the
// pending request IDs
func requestsFromTable(h string) (complete bool, ids []int, err error) {

	// before return, check that if the requests have a response and truncate to only the pending.
	defer func() {
		pendingIds := make([]int, 0)
		for _, pid := range ids {
			p, e := isPending(pid)
			if e != nil {
				log.Println(e)
			}
			if p {
				pendingIds = append(pendingIds, pid)
			}
		}
		ids = pendingIds
	}()

	// first find the upper bound
	upperGtr, err := Api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqctxts",
		LowerBound: h,
		UpperBound: h,
		Limit:      1,
		KeyType:    "i128",
		Index:      "2",
		JSON:       true,
		Reverse:    true,
	})
	if err != nil {
		return
	}
	upper := make([]onlyRequestId, 0)
	err = json.Unmarshal(upperGtr.Rows, &upper)
	if err != nil {
		return
	}
	if len(upper) == 0 {
		return true, make([]int, 0), nil
	}
	u := upper[0].FioRequestId
	if Verbose {
		log.Printf("highest record is %d for index %s\n", upper[0].FioRequestId, h)
	}

	// now the lower
	lowerGtr, err := Api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqctxts",
		LowerBound: h,
		UpperBound: h,
		Limit:      1,
		KeyType:    "i128",
		Index:      "2",
		JSON:       true,
		Reverse:    false,
	})
	if err != nil {
		return
	}
	lower := make([]onlyRequestId, 0)
	err = json.Unmarshal(lowerGtr.Rows, &lower)
	if err != nil {
		return
	}
	if Verbose {
		log.Printf("lowest record is %d for index %s\n", lower[0].FioRequestId, h)
	}
	if lower[0].FioRequestId == upper[0].FioRequestId {
		return true, []int{int(lower[0].FioRequestId)}, nil
	}

	// under normal circumstances we can safely get 500 rows
	// but this is a complete guess, the request id is global not specific to the address,
	// to be safe this assumes worst-case and that all the requests belong to the same address
	if upper[0].FioRequestId-lower[0].FioRequestId <= 500 {
		if Verbose {
			log.Println("attempting one-shot query, less than 500 spread between IDs")
		}
		oneShot := &eos.GetTableRowsResp{}
		oneShot, err = Api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
			Code:       "fio.reqobt",
			Scope:      "fio.reqobt",
			Table:      "fioreqctxts",
			LowerBound: h,
			UpperBound: h,
			Limit:      uint32(upper[0].FioRequestId-lower[0].FioRequestId) + 1,
			KeyType:    "i128",
			Index:      "2",
			JSON:       true,
			Reverse:    false,
		})
		if err != nil {
			return
		}
		once := make([]onlyRequestId, 0)
		err = json.Unmarshal(oneShot.Rows, &once)
		if err != nil {
			return
		}
		// everything there?
		if once != nil && once[0].FioRequestId == lower[0].FioRequestId && once[len(once)-1].FioRequestId == u {
			complete = true
			if Verbose {
				log.Println("got a complete result for ", h)
			}
		}
		ids = make([]int, len(once))
		for i := range once {
			ids[i] = int(once[i].FioRequestId)
		}
		return
	}

	// ok, now we're in the worst case where there's a possibility of having too many records, best effort from here
	// depending on the server speed we can expect between 500-800 rows. Because we are using a secondary index there
	// is no paging. The only alternative is to scan the entire table and look for matches, which will not be feasible
	// as the request table grows.
	split := (uint32(upper[0].FioRequestId-lower[0].FioRequestId) / 2) + 3
	unique := make(map[uint64]bool)
	lowerGtr, err = Api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqctxts",
		LowerBound: h,
		UpperBound: h,
		Limit:      split,
		KeyType:    "i128",
		Index:      "2",
		JSON:       true,
		Reverse:    false,
	})
	if err != nil {
		return
	}
	lower = make([]onlyRequestId, 0)
	err = json.Unmarshal(lowerGtr.Rows, &lower)
	if err != nil {
		return
	}
	if Verbose {
		log.Printf("highest record is %d for ascending search %s\n", lower[len(lower)-1].FioRequestId, h)
	}
	ids = make([]int, len(lower))
	for i, rid := range lower {
		unique[rid.FioRequestId] = true
		ids[i] = int(rid.FioRequestId)
	}
	if ids[len(ids)-1] == int(u) {
		complete = true
		return
	}

	upperGtr, err = Api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqctxts",
		LowerBound: h,
		UpperBound: h,
		Limit:      split,
		KeyType:    "i128",
		Index:      "2",
		JSON:       true,
		Reverse:    true,
	})
	if err != nil {
		return
	}
	upper = make([]onlyRequestId, 0)
	err = json.Unmarshal(upperGtr.Rows, &upper)
	if err != nil || len(upper) == 0 {
		return
	}

	sort.Slice(upper, func(i, j int) bool {
		return upper[i].FioRequestId > upper[j].FioRequestId
	})
	for _, up := range upper {
		// if we overlap, we got it all
		if unique[up.FioRequestId] {
			complete = true
			break
		}
		ids = append(ids, int(up.FioRequestId))
	}
	return
}

func dumpRequests() (ok bool, wrote int, err error) {
	ok = true
	var hashes []string
	hashes, err = getAddrHashes()
	if err != nil || len(hashes) == 0 {
		return
	}
	ids := make([]int, 0)
	missing := false
	for _, h := range hashes {
		var mmmk bool
		var reqs []int
		mmmk, reqs, err = requestsFromTable(h)
		if err != nil {
			return
		}
		ids = append(ids, reqs...)
		if !mmmk && len(reqs) > 0 {
			missing = true
		}
	}
	if missing {
		ok = false
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	buf := bytes.NewBuffer(nil)
	defer func() {
		_, err = F.Write(buf.Bytes())
		_ = F.Close()
	}()
	_, _ = buf.WriteString(csvHeader)
	log.Printf("found %d pending requests, decrypting...", len(ids))
	for count, req := range ids {
		if len(ids) > 100 && count%100 == 0 {
			fmt.Print(count, "... ")
		}
		r, err := Api.GetFioRequest(uint64(req))
		if err != nil {
			log.Printf("getting request %d failed. %v\n", req, err)
			continue
		}
		content, err := fio.DecryptContent(Acc, r.PayeeKey, r.Content, fio.ObtRequestType)
		if err != nil {
			if Verbose {
				log.Printf("decrypting request %d failed. %v - continuing anyway\n", r.FioRequestId, err)
			}
			// ensure not nil, we still want to print what we found.
			content = &fio.ObtContentResult{}
			content.Request = &fio.ObtRequestContent{}
		}
		s := fmt.Sprintf(
			`"%d",%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q`+"\n",
			r.FioRequestId,
			r.PayerFioAddress,
			r.PayerKey,
			r.PayeeFioAddress,
			r.PayeeKey,
			content.Request.PayeePublicAddress,
			content.Request.Amount,
			content.Request.ChainCode,
			content.Request.TokenCode,
			content.Request.Memo,
			content.Request.Hash,
			content.Request.OfflineUrl,
		)
		if Verbose {
			fmt.Print(s)
		}
		_, _ = buf.WriteString(s)
		wrote += 1
	}
	if len(ids) > 100 {
		fmt.Println("")
	}
	return ok, wrote, err
}

func options() {
	var nodeos, privKey, memo string
	var unknown bool

	flag.StringVar(&privKey, "k", "", "private key in WIF format, if absent will prompt")
	flag.StringVar(&InFile, "in", "", "file containing FIO request IDs to reject, incompatible with -out, invokes reqobt::rejectfndreq")
	flag.StringVar(&OutFile, "out", "", "file to dump all outstanding FIO requests into, will be in .CSV format and include decrypted request details")
	flag.StringVar(&nodeos, "u", "https://testnet.fioprotocol.io", "FIO API endpoint to use")
	flag.StringVar(&memo, "memo", "", "memo to send with responses, does not apply to bulk-reject or nuke only with 'confirm'")
	flag.BoolVar(&Confirm, "confirm", false, "true sends a 'recordobt' response, false sends a 'rejectfndreq' only applies with '-in' option")
	flag.BoolVar(&Nuke, "nuke", false, "don't print, don't check, nuke all pending requests. Incompatible with -in -out")
	flag.BoolVar(&unknown, "unknown", false, "allow connecting to unknown chain id")
	flag.BoolVar(&Quiet, "y", false, "assume 'yes' to all questions")
	flag.Parse()

	switch true {
	case InFile == "" && OutFile == "" && !Nuke:
		log.Fatal("either '-in' (file with request IDs to reject, one integer per line) or '-out' (location for .csv report) is required. Use '-h' to list command options")
	case InFile != "" && OutFile != "":
		log.Fatal("only one operation is supported, either '-in' or '-out'. Use '-h' to list command options")
	case OutFile != "":
		Query = true
	}

	_, err := url.Parse(nodeos)
	if err != nil {
		log.Fatal("invalid API host: " + err.Error())
	}

	if privKey == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("please enter the private key: ")
		b, _, err := reader.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		privKey = string(b)
	}

	Acc, Api, _, err = fio.NewWifConnect(privKey, nodeos)
	if err != nil {
		log.Fatal(err)
	}

	gi, err := Api.GetInfo()
	if err != nil {
		log.Fatal(err)
	}
	if gi.HeadBlockTime.Time.After(time.Now().UTC().Add(30 * time.Second)) {
		log.Printf("Head block time (%v) is after the default transaction timeout of 30s.", gi.HeadBlockTime.Time)
		log.Fatal("Is your clock synced?")
	}
	switch gi.ChainID.String() {
	case fio.ChainIdMainnet:
		log.Println("connected to FIO mainnet")
	case fio.ChainIdTestnet:
		log.Println("connected to FIO testnet")
	default:
		if !unknown {
			log.Fatal("refusing to connect to unknown chain id (not mainnet or testnet) override with'-unknown'")
		}
	}

	if os.Getenv("DEBUG") != "" {
		Verbose = true
	}

	if Query {
		F, err = os.OpenFile(OutFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if InFile != "" {
		F, err = os.OpenFile(InFile, os.O_RDONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
}
