package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx := context.WithValue(context.Background(), "state", &feeState{
		FeeVotes:  make([]*fio.FeeVote2, 0),
		FeeVoters: make([]*fio.FeeVoter, 0),
		Producers: make([]*fio.Producer, 0),
	})

	// update state data in the background
	go updateWorker(ctx)

	select {}
}

// updateWorker asynchronously fires various functions that update state data
func updateWorker(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-t.C:
			// DELETEME: debug
			j, _ := json.MarshalIndent(ctx.Value("state"), "", "  ")
			fmt.Println(string(j))

			if !ctx.Value("state").(*feeState).priceBusy {
				go func() {
					if e := updatePrice(ctx); e != nil {
						log.Println("ERROR: could not update price", e)
					}
				}()
			}
			if !ctx.Value("state").(*feeState).votes2Busy {
				go func() {
					if e := updateFeeVotes(ctx); e != nil {
						log.Println("ERROR: could not update feevotes2", e)
					}
				}()
			}
			if !ctx.Value("state").(*feeState).votersBusy {
				go func() {
					if e := updateFeeVoters(ctx); e != nil {
						log.Println("ERROR: could not update feevoters", e)
					}
				}()
			}
			if !ctx.Value("state").(*feeState).prodBusy {
				go func() {
					if e := updateProducers(ctx); e != nil {
						log.Println("ERROR: could not update producers", e)
					}
				}()
			}
		}
	}
}

// feeState holds the results of the most recent update. This data is refreshed periodically.
type feeState struct {
	// Price is pulled from coingecko API, and averaged vs USDT across exchanges.
	Price        float64   `json:"price"`
	PriceUpdated time.Time `json:"price_updated"`
	priceBusy    bool

	// FeeVotes holds the contents of the fio.fee::feevotes2 table
	FeeVotes        []*fio.FeeVote2 `json:"fee_votes"`
	FeeVotesUpdated time.Time       `json:"fee_votes_updated"`
	votes2Mux       sync.RWMutex
	votes2Busy      bool

	// FeeVoters holds the contents of the fio.fee::feevoters table
	FeeVoters        []*fio.FeeVoter `json:"fee_voters"`
	FeeVotersUpdated time.Time       `json:"fee_voters_updated"`
	votersMux        sync.RWMutex
	votersBusy       bool

	// Producers holds the contents of the eosio::producers table, this is used to map account to the Fio Name.
	Producers        []*fio.Producer `json:"producers"`
	ProducersUpdated time.Time       `json:"producers_updated"`
	prodMux          sync.RWMutex
	prodBusy         bool
}

// ready responds with true if we have data sufficient to provide responses
func (fs feeState) ready(ctx context.Context) bool {
	switch true {
	// is any data more than 5 minutes stale?
	case ctx.Value("state").(*feeState).PriceUpdated.Before(time.Now().UTC().Add(-5 * time.Minute)),
		ctx.Value("state").(*feeState).FeeVotersUpdated.Before(time.Now().UTC().Add(-5 * time.Minute)),
		ctx.Value("state").(*feeState).ProducersUpdated.Before(time.Now().UTC().Add(-5 * time.Minute)),
		ctx.Value("state").(*feeState).FeeVotesUpdated.Before(time.Now().UTC().Add(-5 * time.Minute)):
		return false
	default:
		return true
	}
}

// getApi works through a list of public API endpoints, trying until it finds a working endpoint. This is
// to ensure not any one producer being down breaks the service.
func getApi(workerName string, servers []string) (api *fio.API) {
	info := &eos.InfoResp{}
	var err error

	e := func(err string) {
		log.Println(workerName, err)
		time.Sleep(time.Second)
	}

	for {
		api, _, err = fio.NewConnection(nil, servers[rnd(len(servers))])
		if err != nil {
			e(err.Error())
			continue
		}
		log.Println(workerName, "INFO: selected", api.BaseURL, "api server")
		info, err = api.GetInfo()
		if err != nil {
			e(err.Error())
			continue
		}
		if info.ChainID.String() != fio.ChainIdMainnet {
			e("chain ID did not match")
			continue
		}
		if info.HeadBlockTime.Before(time.Now().UTC().Add(-time.Minute)) {
			e("server was > 1 minute behind head")
			continue
		}
		return api
	}
}

// rnd duplicates the math.rand.Intn() functionality, but uses crypto.rand's superior RNG.
func rnd(i int) int {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return int(binary.LittleEndian.Uint32(b)>>1) % i
}

// coinTicker holds a trimmed down response from the coingecko api
type coinTicker struct {
	LastUpdated time.Time  `json:"last_updated"`
	Tickers     []coinTick `json:"tickers"`
}

type coinTick struct {
	Target string  `json:"target"`
	Last   float64 `json:"last"`
}

// updatePrice populates the current price in our state
func updatePrice(ctx context.Context) error {
	ctx.Value("state").(*feeState).priceBusy = true
	defer func() {
		ctx.Value("state").(*feeState).priceBusy = false
	}()
	ct, e := getGecko()
	if e != nil {
		return e
	}
	avg, e := ct.getAvg()
	if e != nil {
		return e
	}
	// not a very thorough check, but should catch serious problems
	if avg < 0 || (ctx.Value("state").(*feeState).Price != 0 && avg > ctx.Value("state").(*feeState).Price*2) {
		return fmt.Errorf("impossible price from coingecko %f", avg)
	}
	ctx.Value("state").(*feeState).Price = avg
	ctx.Value("state").(*feeState).PriceUpdated = time.Now().UTC()
	return nil
}

func getGecko() (*coinTicker, error) {
	const gecko = `https://api.coingecko.com/api/v3/coins/fio-protocol?localization=false&tickers=true&market_data=false&community_data=false&developer_data=false&sparkline=false`

	resp, err := http.Get(gecko)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	j, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	t := &coinTicker{}
	err = json.Unmarshal(j, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetAvg finds all the current USDT exchange rates and calculates an average price
func (t *coinTicker) getAvg() (float64, error) {
	var total, count float64
	for i := range t.Tickers {
		if t.Tickers[i].Target == "USDT" || t.Tickers[i].Target == "USDC" {
			count += 1
			total += t.Tickers[i].Last
		}
	}
	if count == 0 {
		return 0, errors.New("could not get current prices")
	}
	return total / count, nil
}

// updateFeeVotes fetches the current feevotes2 table and stores it in state.
func updateFeeVotes(ctx context.Context) error {
	ctx.Value("state").(*feeState).votes2Busy = true
	defer func() {
		ctx.Value("state").(*feeState).votes2Busy = false
	}()
	mux := &ctx.Value("state").(*feeState).votes2Mux
	mux.Lock()
	defer mux.Unlock()
	api := getApi("updateFeeVotes", producers)
	gtr, err := api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:  "fio.fee",
		Scope: "fio.fee",
		Table: "feevotes2",
		Limit: 1000,
		JSON:  true,
	})
	if err != nil {
		return err
	}
	result := make([]*fio.FeeVote2, 0)
	err = json.Unmarshal(gtr.Rows, &result)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return errors.New("empty query response from updateFeeVotes")
	}
	ctx.Value("state").(*feeState).FeeVotes = result
	ctx.Value("state").(*feeState).FeeVotesUpdated = time.Now().UTC()
	return nil
}

// feeVoterString is an intermediary structure for handling the query response.
// Not sure what's going on here, FeeMultiplier should be a float but getting a string?!?
type feeVoterString struct {
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	FeeMultiplier     string          `json:"fee_multiplier"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}

func (fvs feeVoterString) toFeeVoter() *fio.FeeVoter {
	m, _ := strconv.ParseFloat(fvs.FeeMultiplier, 64)
	return &fio.FeeVoter{
		BlockProducerName: fvs.BlockProducerName,
		FeeMultiplier:     m,
		LastVoteTimestamp: fvs.LastVoteTimestamp,
	}
}

// updateFeeVoters fetches the current feevoters table and stores it in state.
func updateFeeVoters(ctx context.Context) error {
	ctx.Value("state").(*feeState).votersBusy = true
	defer func() {
		ctx.Value("state").(*feeState).votersBusy = false
	}()
	mux := &ctx.Value("state").(*feeState).votersMux
	mux.Lock()
	defer mux.Unlock()
	api := getApi("updateFeeVoters", producers)
	gtr, err := api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:  "fio.fee",
		Scope: "fio.fee",
		Table: "feevoters",
		Limit: 1000,
		JSON:  true,
	})
	if err != nil {
		return err
	}
	result := make([]*feeVoterString, 0)
	err = json.Unmarshal(gtr.Rows, &result)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return errors.New("empty query response from updateFeeVoters")
	}
	fv := make([]*fio.FeeVoter, len(result))
	for i := range result {
		fv[i] = result[i].toFeeVoter()
	}
	ctx.Value("state").(*feeState).FeeVoters = fv
	ctx.Value("state").(*feeState).FeeVotersUpdated = time.Now().UTC()
	return nil
}

// updateProducers fetches the current Producers table and stores it in state.
func updateProducers(ctx context.Context) error {
	ctx.Value("state").(*feeState).prodBusy = true
	defer func() {
		ctx.Value("state").(*feeState).prodBusy = false
	}()
	mux := &ctx.Value("state").(*feeState).prodMux
	mux.Lock()
	defer mux.Unlock()
	api := getApi("updateProducers", producers)
	gtr, err := api.GetTableRowsOrder(fio.GetTableRowsOrderRequest{
		Code:  "eosio",
		Scope: "eosio",
		Table: "producers",
		Limit: 1000,
		JSON:  true,
	})
	if err != nil {
		return err
	}
	result := make([]*fio.Producer, 0)
	err = json.Unmarshal(gtr.Rows, &result)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return errors.New("empty query response from updateProducers")
	}
	ctx.Value("state").(*feeState).Producers = result
	ctx.Value("state").(*feeState).ProducersUpdated = time.Now().UTC()
	return nil
}
