package bulk

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"net/url"
	"os"
	"time"
)

const csvHeader = `"timestamp_utc","request_id","payer","payer_fio","payee","payee_fio","address","amount","chain","token","memo","hash","url"` + "\n"

var (
	InFile  string
	OutFile string
	Name    string
	Memo    string

	Acc *fio.Account
	Api *fio.API
	F   *os.File

	Quiet   bool
	Confirm bool
	Nuke    bool
	Verbose bool
	Query   bool
	SendFio float64
)

func Options() {
	var nodeos, privKey string
	var unknown bool

	flag.StringVar(&privKey, "k", "", "private key in WIF format, if absent will prompt")
	flag.StringVar(&InFile, "in", "", "file containing FIO request IDs to reject, incompatible with -out, invokes reqobt::rejectfndreq")
	flag.StringVar(&OutFile, "out", "", "file to dump all outstanding FIO requests into, will be in .CSV format and include decrypted request details")
	flag.StringVar(&Name, "name", "", "FIO name for requests (required)")
	flag.StringVar(&nodeos, "u", "https://testnet.fioprotocol.io", "FIO API endpoint to use")
	flag.StringVar(&Memo, "memo", "", "memo to send with responses, does not apply to rejected requests: only with '-record'")
	flag.BoolVar(&Confirm, "record", false, "true sends a 'recordobt' response, false sends a 'rejectfndreq' only applies with '-in' option")
	flag.BoolVar(&Nuke, "nuke", false, "don't print, don't check, nuke all pending requests. Incompatible with -in -out")
	flag.BoolVar(&unknown, "unknown", false, "allow connecting to unknown chain id")
	flag.BoolVar(&Quiet, "y", false, "assume 'yes' to all questions")
	flag.Float64Var(&SendFio, "fio", 0.0, "amount of whole FIO (as a float) to send via 'trnsfiopubky' with each response (only applies if '-record=true')")
	flag.Parse()

	if Name == "" {
		log.Fatal("FIO address '-name' option is missing")
	}

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
