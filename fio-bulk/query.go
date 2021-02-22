package bulk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"sort"
	"strconv"
)

// IsPending looks at the fioreqstss and determines if a response has already been sent for the request.
func IsPending(id int) (bool, error) {
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

// GetAddrHashes returns a slice of i128 hashes for all the FIO addresses owned by the account,  which is a partial sha1 sum.
// in order to get all requests using a table lookup, it's necessary to know the address hash so we can query by a secondary key.
func GetAddrHashes() ([]string, error) {
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

// RequestsFromTable uses a table query to (attempt to) bypass the limitation of the API endpoint where it will timeout when
// there are thousands of pending requests, it expects an i128 hash, and returns a slice of int64 representing the
// pending request IDs
func RequestsFromTable(h string) (complete bool, ids []int, err error) {

	// before return, check that if the requests have a response and truncate to only the pending.
	defer func() {
		pendingIds := make([]int, 0)
		for _, pid := range ids {
			p, e := IsPending(pid)
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

func DumpRequests() (ok bool, wrote int, err error) {
	ok = true
	var hashes []string
	hashes, err = GetAddrHashes()
	if err != nil || len(hashes) == 0 {
		return
	}
	ids := make([]int, 0)
	missing := false
	for _, h := range hashes {
		var mmmk bool
		var reqs []int
		mmmk, reqs, err = RequestsFromTable(h)
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
			`"%s",%d",%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q`+"\n",
			r.Time.String(),
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
