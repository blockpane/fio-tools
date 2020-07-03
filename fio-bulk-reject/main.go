package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	feos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const csvHeader = `"request_id","payer","payer_fio","payee","payee_fio","address","amount","chain","token","memo","hash","url"` + "\n"

var (
	inFile  string
	outFile string

	acc *fio.Account
	api *fio.API

	f     *os.File
	query bool
	verbose bool
)

func main() {
	options()
	if query {
		wrote, err := dumpRequests()
		log.Printf("wrote %d records to %s\n", wrote, outFile)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	rejected, err := rejectRequests()
	log.Printf("rejected %d requests from %s\n", rejected, inFile)
	if err != nil {
		log.Fatal(err)
	}
}

func rejectRequests() (rejected int, err error) {
	requests := make([]string, 0)
	reader := bufio.NewReader(f)
	defer f.Close()
	var e error
	var l []byte
	for e == nil {
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
		requests = append(requests, strconv.Itoa(id))
	}
	if verbose {
		fmt.Println(requests)
	}
	for _, id := range requests {
		resp := &feos.PushTransactionFullResp{}
		resp, err = api.SignPushActions(fio.NewRejectFndReq(acc.Actor, id))
		if err != nil {
			return
		}
		log.Printf("rejected %s with txid %s\n", id, resp.TransactionID)
		rejected += 1
	}
	return
}

func dumpRequests() (wrote int, err error) {
	buf := bytes.NewBuffer(nil)
	defer func() {
		_, err = f.Write(buf.Bytes())
		_ = f.Close()
	}()
	_, _ = buf.WriteString(csvHeader)
	offset := 0
	limit := 100
	more := true
More:
	for more {
		requests, found, err := api.GetPendingFioRequests(acc.PubKey, limit, offset)
		if err != nil {
			return 0, err
		}

		switch true {
		case !found && offset == 0:
			var n int
			n, _, err = acc.GetNames(api)
			if err != nil {
				return wrote, err
			}
			user := string(acc.Actor)
			if n > 0 {
				user = acc.Addresses[0].FioAddress
			}
			log.Printf("%s has no pending requests\n", user)
		case !found || len(requests.Requests) == 0:
			more = false
			break More
		case requests.More == 0:
			more = false
		}

		offset += 100
		if requests.More < 100 {
			limit = requests.More
		}

		for _, req := range requests.Requests {
			r, err := api.GetFioRequest(req.FioRequestId)
			if err != nil {
				log.Printf("getting request %d failed. %v\n", req.FioRequestId, err)
				continue
			}
			content, err := fio.DecryptContent(acc, req.PayeeFioPublicKey, r.Content, fio.ObtRequestType)
			if err != nil {
				log.Printf("decrypting request %d failed. %v\n", req.FioRequestId, err)
				continue
			}
			s := fmt.Sprintf(
				`"%d",%q,%q,%q,%q,%q,%q,%q,%q,%q,%q`+"\n",
				req.FioRequestId,
				req.PayerFioAddress,
				req.PayerFioPublicKey,
				req.PayeeFioAddress,
				req.PayeeFioPublicKey,
				content.Request.PayeePublicAddress,
				content.Request.Amount,
				content.Request.ChainCode,
				content.Request.TokenCode,
				content.Request.Memo,
				content.Request.OfflineUrl,
			)
			if verbose {
				fmt.Print(s)
			}
			_, _ = buf.WriteString(s)
			wrote += 1
		}
	}
	return wrote, err
}

func options() {
	var nodeos, privKey string

	flag.StringVar(&privKey, "k", "", "private key in WIF format, if absent will prompt")
	flag.StringVar(&inFile, "in", "", "file containing FIO request IDs to reject, incompatible with -out, invokes reqobt::rejectfndreq")
	flag.StringVar(&outFile, "out", "", "file to dump all outstanding FIO requests into, will be in .CSV format and include decrypted request details")
	flag.StringVar(&nodeos, "h", "https://testnet.fioprotocol.io", "FIO API endpoint to use")
	flag.Parse()

	switch true {
	case inFile == "" && outFile == "":
		log.Fatal("either '-in' (file with request IDs to reject, one integer per line) or '-out' (location for .csv report) is required. Use '-h' to list command options")
	case inFile != "" && outFile != "":
		log.Fatal("only one operation is supported, either '-in' or '-out'. Use '-h' to list command options")
	case outFile != "":
		query = true
		fallthrough
	case !strings.HasPrefix(nodeos, "http"):
		nodeos = "http://" + nodeos
	}

	_, err := url.Parse(nodeos)
	if err != nil {
		log.Fatal("invalid API host: " + err.Error())
	}

	if privKey == "" {
		reader := bufio.NewReader(os.Stdin)
		b, _, err := reader.ReadLine()
		fmt.Print("please enter the private key: ")
		if err != nil {
			log.Fatal(err)
		}
		privKey = string(b)
	}

	acc, api, _, err = fio.NewWifConnect(privKey, nodeos)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("DEBUG") != "" {
		verbose = true
	}

	if query {
		f, err = os.OpenFile(outFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	f, err = os.OpenFile(inFile, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

