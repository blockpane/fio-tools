package main

import (
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
	"sort"
	"strconv"
	"time"
)

const gecko = `https://api.coingecko.com/api/v3/coins/fio-protocol?localization=false&tickers=true&market_data=false&community_data=false&developer_data=false&sparkline=false`

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// this allows running as either a command line function or as an AWS Lambda function:
	// if running as a lambda, use the env vars to set options, preferably using encrypted params to pass in the WIF
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handler)
		return
	}
	if e := handler(); e != nil {
		trace := log.Output(2, e.Error())
		log.Fatal(trace)
	}
}

func handler() error {
	var a, p, wif, nodeos, sTarget string
	flag.StringVar(&a, "actor", "", "optional: account to use for delegated permission, or ACTOR env var")
	flag.StringVar(&p, "permission", "", "optional: permission to use for delegated permission, or PERM env var")
	flag.StringVar(&wif, "wif", "", "required: private key, or WIF env var")
	flag.StringVar(&nodeos, "url", "https://fio.blockpane.com", "optional: nodeos api url, or URL env var")
	flag.StringVar(&sTarget, "target", "2.0", "optional: target price of regaddress in USDC, or TARGET env var")
	flag.Parse()

	if a == ""{
		a = os.Getenv("ACTOR")
	}
	if p == "" {
		p = os.Getenv("PERM")
	}
	if wif == "" {
		wif = os.Getenv("WIF")
	}
	if nodeos == "" {
		nodeos = os.Getenv("URL")
	}
	if sTarget == "" {
		sTarget = os.Getenv("TARGET")
	}

	if wif == "" || nodeos == "" {
		log.Println("Missing NODEOS or WIF environment variable.")
		fmt.Print("\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	acc, api, opt, err := fio.NewWifConnect(wif, nodeos)
	if err != nil {
		return err
	}

	actor := eos.AccountName(a)
	perm := eos.PermissionName(p)

	if string(actor) == "" {
		actor = acc.Actor
	}
	if string(perm) == "" {
		perm = "active"
	}
	if sTarget == "" {
		sTarget = "2.0"
	}

	target, err := strconv.ParseFloat(sTarget, 64)
	if err != nil {
		log.Println("could not parse target price")
		return err
	}

	if update := needsBaseFees(actor, api); update != nil {
		// may help to compress given the size of the request.
		opt.Compress = fio.CompressionZlib
		_, err := api.SignPushActionsWithOpts([]*eos.Action{fio.NewSetFeeVote(update, acc.Actor).ToEos()}, &opt.TxOptions)
		if err != nil {
			log.Println(err)
			log.Println("Could not update base fees, has it been an hour? Continuing anyway")
		}
	}

	prices := getGecko()
	if prices.LastUpdated.Before(time.Now().Add(-1 * time.Hour)) {
		log.Fatal("coingecko data was stale, aborting")
	}
	avg, err := prices.GetAvg()
	if err != nil {
		return err
	}
	setMultiplier := func() {
		prices := getGecko()
		if prices.LastUpdated.Before(time.Now().Add(-1 * time.Hour)) {
			log.Println("coingecko data was stale, aborting")
			return
		}
		avg, err := prices.GetAvg()
		if err != nil {
			log.Println(err)
			return
		}

		defaultFee := float64(getRegFioAddrCost()) / 1_000_000_000.0
		multiplier := target / (defaultFee * avg)

		current, err := GetCurMult(actor, api)
		if err != nil {
			log.Println(err)
			return
		}
		if math.Abs(current-multiplier) > 0.1 {
			if current != 0 && (math.Abs(current-multiplier)/current > 0.15) {
				log.Println("current fee is:", current, "proposed fee is:", multiplier)
				log.Fatal("New fee multiplier would be more than a 15% change, please set it manually to continue automatically adjusting fees")
			}

			act := fio.NewActionWithPermission("fio.fee", "setfeemult", actor, string(perm), fio.SetFeeMult{
				Multiplier: multiplier,
				Actor:      actor,
				MaxFee:     fio.Tokens(fio.GetMaxFee(fio.FeeSubmitFeeMult)),
			})
			resp, err := api.SignPushActions(act)
			if err != nil {
				log.Println(resp)
				log.Println(err)
				// don't bail, try the ComputeFees call on the way out
			}
		} else {
			log.Printf("Fee has not changed enough to re-submit: existing %f, proposed %f\n", current, multiplier)
			return
		}

		// this can fail silently
		_, err = api.SignPushActions(fio.NewActionWithPermission("fio.fee", "computefees", actor, string(perm), fio.ComputeFees{}))
		if err != nil {
			log.Println("Compute fees failed (can safely ignore): ", err.Error())
		}
	} else {
		log.Println("Fee has not changed enough to re-submit")
		return nil
	}

	setMultiplier()
	ticker := time.NewTicker(60 * time.Minute)
	for {
		select {
		case <-ticker.C:
			setMultiplier()
		}
	}
	return nil
}

func defaultFee() []*fio.FeeValue {
	// FIXME: this smells. Should it be in a config file? Makes running as an AWS lambda harder, rethink it later.
	defaults := []*fio.FeeValue{
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
		{EndPoint: "submit_fee_multiplier", Value: 60000000},
		{EndPoint: "submit_fee_ratios", Value: 5000000}, // override default: if we call this often, want a low price
		{EndPoint: "transfer_fio_address", Value: 60000000},
		{EndPoint: "transfer_fio_domain", Value: 100000000},
		{EndPoint: "transfer_tokens_pub_key", Value: 100000000},
		{EndPoint: "unregister_producer", Value: 20000000},
		{EndPoint: "unregister_proxy", Value: 20000000},
		{EndPoint: "vote_producer", Value: 30000000},
	}
	return defaults
}

func getRegFioAddrCost() uint64 {
	fees := defaultFee()
	for i := range fees {
		if fees[i].EndPoint == "register_fio_address" {
			return uint64(fees[i].Value)
		}
	}
	log.Fatal("could not determine default value for register_fio_address, aborting")
	return 0 // unreachable, but don't let compiler complain
}

// needsBaseFees checks the current feevotes2 table and returns a nil if fees are set as expected.
// otherwise, the returned value should be submitted.
func needsBaseFees(actor eos.AccountName, api *fio.API) (proposed []*fio.FeeValue) {
	defaults := defaultFee()

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
		panic(err)
	}
	type ev struct {
		Feevotes []struct{
			EndPoint string `json:"end_point"`
			Value int64 `json:"value"`
		} `json:"feevotes"`
	}
	maybeBlanks := make([]ev, 0)
	err = json.Unmarshal(gtr.Rows, &maybeBlanks)
	if err != nil {
		panic(err)
	}
	existing := make([]fio.FeeValue, 0)
	for _, v := range maybeBlanks[0].Feevotes {
		if v.EndPoint == "" || v.Value < 0 {
			continue
		}
		existing = append(existing, fio.FeeValue{EndPoint: v.EndPoint, Value: uint64(v.Value)})
	}
	sort.Slice(existing, func(i, j int) bool {
		return defaults[i].EndPoint < defaults[j].EndPoint
	})
	if len(existing) != len(defaults) {
		log.Printf("different number of fee values on-chain (%d) vs desired (%d)", len(existing), len(defaults))
		return defaults
	}
	for i := range existing {
		var sendDefault bool
		if existing[i].EndPoint != defaults[i].EndPoint || existing[i].Value != defaults[i].Value {
			log.Println("on-chain data differs for fee endpoint:", existing[i].EndPoint)
			sendDefault = true
		}
		if sendDefault {
			return defaults
		}
	}
	return nil
}

// ticker holds a trimmed down response from the coingecko api
type ticker struct {
	LastUpdated time.Time `json:"last_updated"`
	Tickers     []tick    `json:"tickers"`
}

type tick struct {
	Target string  `json:"target"`
	Last   float64 `json:"last"`
}

func getGecko() *ticker {
	resp, err := http.Get(gecko)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	j, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	t := &ticker{}
	err = json.Unmarshal(j, t)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

// GetAvg finds all the current USDT exchange rates and calculates an average price
func (t *ticker) GetAvg() (float64, error) {
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
