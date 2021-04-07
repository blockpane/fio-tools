package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const gecko = `https://api.coingecko.com/api/v3/coins/fio-protocol?localization=false&tickers=true&market_data=false&community_data=false&developer_data=false&sparkline=false`

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// this allows running as either a daemon or as an AWS Lambda function:
	// if running as a lambda, use the env vars to set options, preferably using encrypted SSM params to pass in the WIF
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handler)
		return
	}
	if e := handler(); e != nil {
		_ = log.Output(2, e.Error())
		os.Exit(1)
	}
}

func handler() error {
	var a, p, wif, nodeos, sTarget, customFees, myName string
	var frequency int
	var once, claim, skip, simulate, example bool
	flag.StringVar(&a, "actor", "", "optional: account to use for delegated permission, alternate: ACTOR env var")
	flag.StringVar(&p, "permission", "", "optional: permission to use for delegated permission, alternate: PERM env var")
	flag.StringVar(&wif, "wif", "", "required: private key, alternate: WIF env var")
	flag.StringVar(&nodeos, "url", "", "required: nodeos api url, alternate: URL env var")
	flag.StringVar(&sTarget, "target", "2.0", "optional: target price of regaddress in USDC, alternate: TARGET env var")
	flag.StringVar(&customFees, "fees", "", "optional: JSON file for overriding default fee votes, alternate: JSON env var")
	flag.IntVar(&frequency, "frequency", 2, "optional: hours to wait between runs (does not apply to AWS Lambda), alternate FREQ env var")
	flag.BoolVar(&once, "x", false, "optional: exit after running once (does not apply to AWS Lambda,) use for running from cron")
	flag.BoolVar(&claim, "claim", false, "optional: perform tpidclaim and bpclaim each run, alternate: CLAIM env var")
	flag.BoolVar(&skip, "skip", false, "optional: skip feevote (only do feemult votes) alternate: SKIP env var")
	flag.BoolVar(&simulate, "simulate", false, "optional: do not send any transactions, only print what would have been done, alternate: SIMULATE env var")
	flag.BoolVar(&example, "example", false, "print out the default fees that fio-fee-vote would use and exit.")
	flag.StringVar(&myName, "name", "", "optional: FIO name to be used when performing bpclaim and tpidclaim (required when -claim=true), alternate: NAME env var")
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println(`
Example of json input format for '--fees' flag / JSON env var:

[
  {
    "end_point": "add_pub_address",
    "value": 30000000
  },
  {
    "end_point": "add_to_whitelist",
    "value": 30000000
  }
]`)
	}
	flag.Parse()

	if example {
		j, _ := json.MarshalIndent(defaultFee(), "", "  ")
		fmt.Println(string(j))
		os.Exit(0)
	}

	if strings.ToLower(os.Getenv("SIMULATE")) == "true" {
		simulate = true
	}
	if simulate {
		log.Println("Simulation mode, no transactions will be sent, actions will be printed.")
	}

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		once = true
	}

	if !claim && os.Getenv("CLAIM") != "" {
		claim = true
	}
	if !skip && os.Getenv("SKIP") != "" {
		skip = true
	}

	if os.Getenv("FREQ") != "" {
		dur, err := strconv.ParseInt(os.Getenv("FREQ"), 10, 32)
		if err == nil && dur > 0 {
			frequency = int(dur)
			log.Println("set frequency via ENV to:", frequency)
		}
	}

	switch "" {
	case a:
		a = os.Getenv("ACTOR")
		fallthrough
	case p:
		p = os.Getenv("PERM")
		fallthrough
	case wif:
		wif = os.Getenv("WIF")
		fallthrough
	case nodeos:
		nodeos = os.Getenv("URL")
		fallthrough
	case sTarget:
		sTarget = os.Getenv("TARGET")
		fallthrough
	case myName:
		myName = os.Getenv("NAME")
		fallthrough
	case customFees:
		customFees = os.Getenv("JSON")
	}

	if wif == "" || nodeos == "" {
		fmt.Print("\nOptions:\n")
		flag.PrintDefaults()
		return errors.New("missing URL or WIF environment variable")
	}

	acc, api, opt, err := fio.NewWifConnect(wif, nodeos)
	if err != nil {
		return err
	}

	actor := eos.AccountName(a)
	perm := eos.PermissionName(p)

	switch "" {
	case string(actor):
		actor = acc.Actor
		fallthrough
	case string(perm):
		perm = "active"
		fallthrough
	case sTarget:
		sTarget = "2.0"
	}

	target, err := strconv.ParseFloat(sTarget, 64)
	if err != nil {
		log.Println("could not parse target price")
		return err
	}

	update := make([]*fio.FeeValue, 0)
	switch "" {
	case customFees:
		if skip {
			break
		}
		update, err = needsBaseFees(nil, actor, api)
		if err != nil && once {
			return err
		}
	default:
		if skip {
			break
		}
		custom := make([]*fio.FeeValue, 0)
		var f *os.File
		f, err = os.OpenFile(customFees, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("could not open custom fees")
			return err
		}
		var j []byte
		j, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		_ = f.Close()
		err = json.Unmarshal(j, &custom)
		if err != nil {
			log.Println("could not parse custom fees")
			return err
		}
		if len(custom) == 0 {
			log.Println("WARNING: json file provided had no entries, or was not in the correct format. Will not attempt fee updates.")
			skip = true
			break
		}
		update, err = needsBaseFees(custom, actor, api)
		if err != nil && once {
			return err
		}
	}

	for {
		if ok, _ := isProducer(actor, claim, myName, api); !ok && !simulate {
			log.Println("not an active producer, or other problems, waiting to set fees until registered")
			time.Sleep(10 * time.Minute)
			continue
		}
		if update != nil && !skip {
			log.Println("attempting feevote update")
			api.RefreshFees()
			// may help to compress given the size of the request.
			opt.Compress = fio.CompressionZlib
			act := fio.SetFeeVote{FeeRatios: update, MaxFee: fio.Tokens(fio.GetMaxFeeByAction("setfeevote")), Actor: actor}
			switch simulate {
			case true:
				printAction(act)
				err = nil
			default:
				_, err = api.SignPushActionsWithOpts([]*eos.Action{fio.NewActionWithPermission("fio.fee", "setfeevote", actor, string(perm), act).ToEos()}, &opt.TxOptions)
			}
			if err != nil {
				log.Println(err)
				log.Println("Could not update base fees, has it been an hour? Continuing anyway")
			} else {
				log.Println("feevote updated")
			}
		}
		break
	}

	setMultiplier := func() error {

		// maint is a few maintenance calls all BPs should be calling to trigger fee updates, also adding a burnexpired to cleanup
		// addresses that should be removed from state
		maint := func() {
			if simulate {
				log.Println("would have sent fio.fee::computefees and fio.address::burnexpired")
				return
			}
			// this can fail without consequence, try to call it several times across multiple blocks.
			for i := 0; i < 3; i++ {
				_, err = api.SignPushActions(fio.NewActionWithPermission("fio.fee", "computefees", actor, string(perm), fio.ComputeFees{}))
				if err != nil {
					log.Println("Compute fees failed (can safely ignore): ", err.Error())
					break
				}
				time.Sleep(time.Second)
			}
			// throw in a quick burnexpired for good measure, this won't even be possible until after late March 2021
			// when addresses start expiring.
			_, err = api.SignPushActions(fio.NewActionWithPermission("fio.address", "burnexpired", actor, string(perm), fio.BurnExpired{}))
			if err != nil {
				log.Println("Burn expired failed (can safely ignore): ", err.Error())
			}
		}
		// call the maintenance calls on the way out everytime, even if we didn't set fees/multiplier.
		defer maint()

		var prices *coinTicker
		var avg, defFee, current float64
		var df uint64

		prices, err = getGecko()
		if err != nil {
			return err
		}
		if prices.LastUpdated.Before(time.Now().Add(-1 * time.Hour)) {
			return errors.New("coingecko data was stale, aborting")
		}
		avg, err = prices.GetAvg()
		if err != nil {
			return err
		}

		df, err = getRegFioAddrCost()
		if err != nil {
			return err
		}
		defFee = float64(df) / 1_000_000_000.0
		multiplier := target / (defFee * avg)

		current, err = GetCurMult(actor, api)
		if err != nil {
			return err
		}
		// don't submit for tiny changes
		if math.Abs(current-multiplier) > 0.15 {
			// don't submit for huge changes
			if current != 0 && (math.Abs(current-multiplier)/current > 0.25) {
				log.Println("current fee is:", current, "proposed fee is:", multiplier)
				return errors.New("new fee multiplier would be more than a 25% change, please set it manually to continue automatically adjusting fees")
			}

			api.RefreshFees() // ensure we don't underpay if running as a daemon
			act := fio.NewActionWithPermission("fio.fee", "setfeemult", actor, string(perm), fio.SetFeeMult{
				Multiplier: multiplier,
				Actor:      actor,
				MaxFee:     fio.Tokens(fio.GetMaxFee(fio.FeeSubmitFeeMult)),
			})
			switch simulate {
			case true:
				printAction(act)
			default:
				_, err = api.SignPushActions(act)
				if err != nil {
					log.Println("Setting fees failed:", err)
					// don't bail, try the ComputeFees call on the way out
				}
			}
		} else {
			log.Printf("Multiplier has not changed enough to re-submit: existing %f, proposed %f\n", current, multiplier)
			return nil
		}

		return nil
	}

	doClaims := func() {
		if !claim || myName == "" {
			return
		}
		if simulate {
			log.Println("would have called fio.treasury::bpclaim and fio.treasury::tpidclaim")
			return
		}
		act := fio.NewActionWithPermission("fio.treasury", "bpclaim", actor, string(perm), fio.BpClaim{
			FioAddress: myName,
			Actor:      actor,
		})
		_, err = api.SignPushActions(act)
		if err != nil {
			log.Println(err)
		}
		act = fio.NewActionWithPermission("fio.treasury", "tpidclaim", actor, string(perm), fio.PayTpidRewards{
			Actor: actor,
		})
		_, err = api.SignPushActions(act)
		if err != nil {
			log.Println(err)
		}
	}

	ok, err := isProducer(actor, claim, myName, api)
	if ok && err == nil {
		doClaims()
		err = setMultiplier()
	}
	// don't loop if running as lambda function
	if once {
		return err
	}
	ticker := time.NewTicker(time.Duration(frequency) * time.Hour)
	for {
		select {
		case <-ticker.C:
			go func() {
				ok, err = isProducer(actor, claim, myName, api)
				if !ok {
					log.Println("problems with registration (is account a registered producer?), sleeping")
					return
				}
				doClaims()
				// add some variability to when this starts, less predictability makes it less likely to be subjected
				// to timing / flash attacks.
				time.Sleep(time.Duration(intRand(10)) * time.Minute)
				if err = setMultiplier(); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

func defaultFee() []*fio.FeeValue {
	defaults := []*fio.FeeValue{
		{EndPoint: "add_bundled_transactions", Value: 2000000000},
		{EndPoint: "add_pub_address", Value: 30000000},
		{EndPoint: "add_to_whitelist", Value: 30000000},
		{EndPoint: "auth_delete", Value: 20000000},
		{EndPoint: "auth_link", Value: 20000000},
		{EndPoint: "auth_update", Value: 50000000},
		{EndPoint: "burn_fio_address", Value: 60000000},
		{EndPoint: "cancel_funds_request", Value: 60000000},
		{EndPoint: "msig_approve", Value: 20000000},
		{EndPoint: "msig_cancel", Value: 20000000},
		{EndPoint: "msig_exec", Value: 20000000},
		{EndPoint: "msig_invalidate", Value: 20000000},
		{EndPoint: "msig_propose", Value: 50000000},
		{EndPoint: "msig_unapprove", Value: 20000000},
		{EndPoint: "new_funds_request", Value: 60000000},
		{EndPoint: "proxy_vote", Value: 30000000},
		{EndPoint: "record_obt_data", Value: 60000000},
		{EndPoint: "register_fio_address", Value: 2000000000},
		{EndPoint: "register_fio_domain", Value: 40000000000},
		{EndPoint: "register_producer", Value: 10000000000},
		{EndPoint: "register_proxy", Value: 1000000000},
		{EndPoint: "reject_funds_request", Value: 30000000},
		{EndPoint: "remove_all_pub_addresses", Value: 60000000},
		{EndPoint: "remove_from_whitelist", Value: 30000000},
		{EndPoint: "remove_pub_address", Value: 60000000},
		{EndPoint: "renew_fio_address", Value: 2000000000},
		{EndPoint: "renew_fio_domain", Value: 40000000000},
		{EndPoint: "set_fio_domain_public", Value: 30000000},
		{EndPoint: "submit_bundled_transaction", Value: 30000000},
		{EndPoint: "submit_fee_multiplier", Value: 10000000}, // I'm overriding the default here, this should be cheap.
		{EndPoint: "submit_fee_ratios", Value: 70000000},
		{EndPoint: "transfer_fio_address", Value: 60000000},
		{EndPoint: "transfer_fio_domain", Value: 100000000},
		{EndPoint: "transfer_locked_tokens", Value: 100000000},
		{EndPoint: "transfer_tokens_pub_key", Value: 100000000},
		{EndPoint: "unregister_producer", Value: 20000000},
		{EndPoint: "unregister_proxy", Value: 20000000},
		{EndPoint: "vote_producer", Value: 30000000},
	}
	return defaults
}

func getRegFioAddrCost() (uint64, error) {
	fees := defaultFee()
	for i := range fees {
		if fees[i].EndPoint == "register_fio_address" {
			return uint64(fees[i].Value), nil
		}
	}
	return 0, errors.New("could not determine default value for register_fio_address, aborting")
}

// needsBaseFees checks the current feevotes2 table and returns a nil if fees are set as expected.
// otherwise, the returned value should be submitted.
func needsBaseFees(fees []*fio.FeeValue, actor eos.AccountName, api *fio.API) (proposed []*fio.FeeValue, err error) {
	if fees == nil || len(fees) == 0 {
		fees = defaultFee()
	}

	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.fee",
		Scope:      "fio.fee",
		Table:      "feevotes2",
		LowerBound: string(actor),
		UpperBound: string(actor),
		Limit:      1,
		KeyType:    "name",
		Index:      "2",
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	type ev struct {
		Feevotes []struct {
			EndPoint string `json:"end_point"`
			Value    int64  `json:"value"`
		} `json:"feevotes"`
	}
	maybeBlanks := make([]ev, 0)
	err = json.Unmarshal(gtr.Rows, &maybeBlanks)
	if err != nil {
		return nil, err
	}
	existing := make([]fio.FeeValue, 0)
	if maybeBlanks == nil || len(maybeBlanks) == 0 || maybeBlanks[0].Feevotes == nil {
		return defaultFee(), nil
	}
	for _, v := range maybeBlanks[0].Feevotes {
		if v.EndPoint == "" || v.Value < 0 {
			continue
		}
		existing = append(existing, fio.FeeValue{EndPoint: v.EndPoint, Value: v.Value})
	}
	if len(existing) != len(fees) {
		log.Printf("different number of fee values on-chain (%d) vs desired (%d)", len(existing), len(fees))
		return fees, nil
	}
	// use a map instead of trying to compare slices
	oldFee, newFee := make(map[string]int64), make(map[string]int64)
	for _, v := range existing {
		oldFee[v.EndPoint] = v.Value
	}
	for _, v := range fees {
		newFee[v.EndPoint] = v.Value
	}

	for k, v := range newFee {
		var sendDefault bool
		if oldFee[k] != v {
			log.Println("on-chain data differs for desired fee endpoint:", k)
			sendDefault = true
		}
		if sendDefault {
			return fees, nil
		}
	}
	return nil, nil
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

func getGecko() (*coinTicker, error) {
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
func (t *coinTicker) GetAvg() (float64, error) {
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

type FeeMultResp struct {
	FeeMultiplier string `json:"fee_multiplier"`
}

func GetCurMult(actor eos.AccountName, api *fio.API) (float64, error) {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.fee",
		Scope:      "fio.fee",
		Table:      "feevoters",
		LowerBound: string(actor),
		UpperBound: string(actor),
		Limit:      1,
		KeyType:    "name",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		return 0, err
	}
	current := make([]FeeMultResp, 0)
	err = json.Unmarshal(gtr.Rows, &current)
	if len(current) == 0 {
		return 0, nil
	}
	fm, err := strconv.ParseFloat(current[0].FeeMultiplier, 64)
	if err != nil {
		return 0, err
	}
	return fm, err
}

type trimmedProd struct {
	Owner      eos.AccountName `json:"owner"`
	FioAddress string          `json:"fio_address"`
	IsActive   uint8           `json:"is_active"`
}

func isProducer(actor eos.AccountName, claim bool, fioName string, api *fio.API) (bool, error) {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: string(actor),
		UpperBound: string(actor),
		Limit:      1,
		KeyType:    "name",
		Index:      "4",
		JSON:       true,
	})
	if err != nil {
		return false, err
	}
	current := make([]trimmedProd, 0)
	err = json.Unmarshal(gtr.Rows, &current)
	if len(current) == 0 {
		log.Println("not registered as an active producer")
		return false, nil
	}
	switch true {
	case current[0].Owner != actor:
		log.Println("got a bad match on account name")
		return false, err
	case current[0].IsActive == 0:
		log.Println("producer is not active")
		return false, err
	case claim && current[0].FioAddress != fioName:
		log.Println("FIO name does not match for producer")
		return false, err
	}
	return true, nil
}

func printAction(v ...interface{}) {
	log.Println("would have sent transaction:")
	j, err := json.MarshalIndent(v, "                    ", "  ")
	if err != nil {
		log.Println(err)
	}
	fmt.Println(string(j))
}

// better than math's rand.Intn which is eerily predictable
func intRand(i int) int {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("could not get bytes from RNG")
		panic(err)
	}
	// strip possible signed bit, cast, and return modulus
	return int(binary.LittleEndian.Uint32(b) >> 1) % i
}
