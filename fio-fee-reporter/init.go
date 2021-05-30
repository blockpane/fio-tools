package ffr

import (
	"github.com/fioprotocol/fio-go"
	"log"
	"os"
	"strconv"
	"strings"
)

var producers = []string{
	"https://api.fio.alohaeos.com",         //  Aloha EOS
	"https://api.fio.currencyhub.io",       //  Currency Hub
	"https://api.fio.eosdetroit.io",        //  EOS Detroit
	"https://api.fio.greeneosio.com",       //  Green EOSIO
	"https://api.fio.services",             //  fio.services
	"https://api.fiosweden.org",            //  EOS Sweden
	"https://fio-bp.dmail.co",              //  Dmail
	"https://fio-mainnet.eosblocksmith.io", //  EOS Blocksmith
	"https://fio-za.eostribe.io",           //  EOS Tribe
	"https://fio.acherontrading.com",       //  Acheron
	"https://fio.blockpane.com",            //  Block Pane
	"https://fio.cryptolions.io",           //  Cryptolions
	"https://fio.eos.barcelona",            //  EOS Barcelona
	"https://fio.eosargentina.io",          //  EOS Argentina
	"https://fio.eoscannon.io",             //  EOS Cannon
	"https://fio.eosdac.io",                //  EOS DAC
	"https://fio.eosdublin.io",             //  EOS Dublin
	"https://fio.eosphere.io",              //  EOSphere
	"https://fio.eosrio.io",                //  EOSRIO
	"https://fio.eosusa.news",              //  EOSUSA
	"https://fio.eu.eosamsterdam.net",      //  EOS Amsterdam
	"https://fio.genereos.io",              //  Genereos
	"https://fio.greymass.com",             //  Greymass
	"https://fio.zenblocks.io",             //  Zen Blocks
	"https://fioapi.ledgerwise.io",         //  Ledgerwise
}

var (
	update int
	srvrs  string
	state  = &feeState{
		FeeVotes:  make([]*fio.FeeVote2, 0),
		FeeVoters: make([]*fio.FeeVoter, 0),
		Producers: make([]*fio.Producer, 0),
	}
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	options()

	// update state data in the background
	go state.updateWorker()
}

// options parses command line flags, or uses env vars for settings.
// FIXME: flags now handled by go-swagger cmd
func options() {
	//flag.StringVar(&srvrs, "servers", "", "optional: list of nodeos servers to use, comma seperated (alt env: SERVERS)")
	//flag.IntVar(&port, "p", 8000, "port to listen on for incoming rest requests (alt env: PORT)")
	//flag.IntVar(&update, "update", 1, "update frequency for source data in minutes (alt env: UPDATE")
	//flag.Parse()
	update = 5

	switch false {
	case os.Getenv("SERVERS") == "":
		srvrs = os.Getenv("SERVERS")
		fallthrough
	//case os.Getenv("PORT") == "":
	//	p, e := strconv.ParseInt(os.Getenv("PORT"), 10, 32)
	//	if e != nil {
	//		log.Fatal("Invalid PORT env var:", e)
	//	}
	//	port = int(p)
	//	fallthrough
	case os.Getenv("UPDATE") == "":
		u, e := strconv.ParseInt(os.Getenv("UPDATE"), 10, 32)
		if e != nil {
			log.Fatal("Invalid UPDATE env var:", e)
		}
		update = int(u)
	}

	if srvrs != "" {
		p := strings.Split(srvrs, ",")
		if len(p) > 0 {
			for i := range p {
				p[i] = strings.TrimSpace(p[i])
				if !strings.HasPrefix(p[i], "http") {
					log.Fatal("invalid server URL:", p[i])
				}
			}
			producers = p
		}
	}
}
