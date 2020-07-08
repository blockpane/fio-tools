package airdrop

import (
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"os"
	"strconv"
	"time"
)

const maxRetries = 3

func init() {
	log.SetPrefix(" [airdrop] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)
}

func Setup() (acc *fio.Account, api *fio.API, tokens float64, reportFile *os.File, dryRun bool){
	var err error
	var nodeos, privKey, tpid, file string
	flag.StringVar(&nodeos, "u", "", "nodoes URL to connect to, env var: NODEOS")
	flag.StringVar(&privKey, "k", "", "WIF key to use, env var: WIF")
	flag.StringVar(&tpid, "t", "", "TPID for transactions, env var: TPID")
	flag.StringVar(&file, "out", "", "filename for saving CSV of results, default stdout, env var: OUT")
	flag.Float64Var(&tokens, "amount", 50.0, "amount to send in airdrop, env var: AMOUNT")
	flag.BoolVar(&dryRun, "dry-run", false, "do not send tokens, only show what would be done")

	flag.Parse()

	if os.Getenv("NODOES") != "" {
		nodeos = os.Getenv("NODEOS")
	}
	if os.Getenv("OUT") != "" {
		file = os.Getenv("OUT")
	}
	if os.Getenv("WIF") != "" {
		privKey = os.Getenv("WIF")
	}
	if os.Getenv("TPID") != "" {
		tpid = os.Getenv("TPID")
	}
	if os.Getenv("AMOUNT") != "" {
		var t float64
		t, err = strconv.ParseFloat(os.Getenv("AMOUNT"), 64)
		if err != nil {
			log.Fatal("invalid value for AMOUNT: "+err.Error())
		}
		tokens = t
	}
	if tokens == 0 {
		log.Fatal("amount for airdrop must be non-zero")
	}

	if file != "" && !dryRun {
		reportFile, err = os.OpenFile(file, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	acc, api, _, err = fio.NewWifConnect(privKey, nodeos)
	if err != nil {
		log.Fatal(err)
	}
	if !api.HasHistory() && !dryRun {
		log.Fatal("This tool requires a v1 history node to confirm transactions after finality")
	}
	log.Println("v1 history API is available")

	if tpid != "" && fio.SetTpid(tpid) {
		log.Println("set TPID to "+tpid)
	}

	gi, err := api.GetInfo()
	if err != nil {
		log.Fatal(err)
	}
	if gi.HeadBlockTime.Time.Before(time.Now().Add(-3*time.Minute)) {
		log.Fatal("head block time is more than 3 minutes behind actual time, is this node syncing?")
	}
	log.Println("node appears to be synced, starting airdrop.")

	if gi.ChainID.String() == fio.ChainIdMainnet {
		fmt.Println("\n***************** WARNING ***************** ")
		fmt.Println("        Mainnet ChainID detected!")
		fmt.Println("       this will spend real tokens")
		fmt.Println("sleeping 10 seconds, press CTRL-C to abort.")
		fmt.Println("***************** WARNING ***************** ")
		fmt.Println("")
		time.Sleep(10*time.Second)
	}
	return
}

