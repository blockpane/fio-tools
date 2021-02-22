package bulk

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"os"
	"strconv"
	"strings"
)

func Respond() (success int, fails int, err error) {
	stat, err := F.Stat()
	if err != nil {
		return 0, 0, err
	}
	if stat.Size() <= 1 {
		return 0, 0, errors.New("empty file")
	}
	requests, err := slurp()
	if err != nil {
		return 0, len(requests), err
	}
	if len(requests) == 0 {
		log.Println("no work to perform")
		os.Exit(0)
	}

	// check for bundles, and prompt if not enough:
	func() {
		ok, need, e := checkBundles(len(requests), bundleRespond)
		if e != nil {
			log.Fatal("cannot lookup bundle, aborting:", e)
		}
		if ok {
			return
		}

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Printf("WARNING: not enough bundles to cover recordobt need %d more, continue anyway? [y/n]: ", need)
			scanner.Scan()
			answer := scanner.Text()
			switch true {
			case strings.HasPrefix(strings.ToLower(answer), "y"):
				return
			case strings.HasPrefix(strings.ToLower(answer), "n"):
				log.Println("aborting!")
				os.Exit(0)
			}
		}
	}()

	// prompt if sending tokens:
	if SendFio > 0 {
		needs, ok, e := checkTransfers(Acc.Actor, len(requests), SendFio)
		if !ok {
			log.Printf("insufficient funds to handle transfer: need %f additional FIO to complete.\n", needs)
			os.Exit(1)
		}
		if e != nil {
			log.Fatal(e)
		}
		if Confirm {
			func() {
				scanner := bufio.NewScanner(os.Stdin)
				for {
					fmt.Printf(
						"WARNING: this will cost %f FIO tokens, continue? [y/n]: ",
						(float64(len(requests))*fio.GetMaxFee(fio.FeeTransferTokensPubKey))+(float64(len(requests))*SendFio),
					)
					scanner.Scan()
					answer := scanner.Text()
					switch true {
					case strings.HasPrefix(strings.ToLower(answer), "y"):
						return
					case strings.HasPrefix(strings.ToLower(answer), "n"):
						log.Println("aborting!")
						os.Exit(0)
					}
				}
			}()
		}
	}

	data := &reqInTable{}
	for _, req := range requests {
		data, err = getRequest(req)
		if err != nil {
			fails += 1
			log.Println("error looking up request id", req, err)
			continue
		}
		var xferId, respId string
		xferId, respId, err = confirmRequest(data)
		if err != nil {
			fails += 1
			log.Println("error responding to request id", req, err)
			continue
		}
		success += 1
		switch SendFio {
		case 0.0:
			log.Printf("responded to request id %s with transaction id %s\n", req, respId)
		default:
			log.Printf("sent %f FIO in txid %s for request id %s, responded with txid %s\n", SendFio, xferId, req, respId)
		}
	}

	return
}

func confirmRequest(req *reqInTable) (xferTxid, respTxid string, err error) {
	// are we transferring tokens first?
	if SendFio > 0 {
		result := &eos.PushTransactionFullResp{}
		result, err = Api.SignPushActions(fio.NewTransferTokensPubKey(Acc.Actor, req.PayeeKey, fio.Tokens(SendFio)))
		if err != nil {
			return "", "", err
		}
		xferTxid = result.TransactionID
	}

	content, err := fio.ObtRecordContent{
		PayerPublicAddress: string(req.PayerFioAddr),
		PayeePublicAddress: string(req.PayeeFioAddr),
		Amount:             fmt.Sprintf("%f", SendFio),
		ChainCode:          "FIO",
		TokenCode:          "FIO",
		ObtId:              strconv.Itoa(req.FioRequestId),
		Memo:               Memo,
		Hash:               xferTxid,
	}.Encrypt(Acc, req.PayeeKey)
	if err != nil {
		return
	}
	result, err := Api.SignPushActions(fio.NewRecordSend(
		Acc.Actor,
		strconv.Itoa(req.FioRequestId),
		string(req.PayerFioAddr),
		string(req.PayeeFioAddr),
		content,
	))
	if err != nil {
		return
	}
	respTxid = result.TransactionID
	return
}

type reqInTable struct {
	FioRequestId int         `json:"fio_request_id"`
	Content      string      `json:"content"`
	TimeStamp    int64       `json:"time_stamp"`
	PayerFioAddr fio.Address `json:"payer_fio_addr"`
	PayeeFioAddr fio.Address `json:"payee_fio_addr"`
	PayerKey     string      `json:"payer_key"`
	PayeeKey     string      `json:"payee_key"`
}

func getRequest(id string) (*reqInTable, error) {
	gtr, err := Api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.reqobt",
		Scope:      "fio.reqobt",
		Table:      "fioreqctxts",
		Index:      "1",
		KeyType:    "i64",
		LowerBound: id,
		UpperBound: id,
		Limit:      1,
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	results := make([]reqInTable, 0)
	err = json.Unmarshal(gtr.Rows, &results)
	if err != nil {
		log.Println("request id:", id, err)
		return nil, err
	}
	if results != nil && len(results) != 1 {
		return nil, errors.New("could not find id " + id + " in fioreqctxts table")
	}
	return &results[0], nil
}
