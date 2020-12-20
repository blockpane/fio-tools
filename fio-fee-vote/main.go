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
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
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
		trace := log.Output(2, e.Error())
		log.Fatal(trace)
	}
}

func handler() error {
	var a, p, wif, nodeos, sTarget, customFees string
	var frequency int
	var once bool
	flag.StringVar(&a, "actor", "", "optional: account to use for delegated permission, alternate: $ACTOR env var")
	flag.StringVar(&p, "permission", "", "optional: permission to use for delegated permission, alternate: $PERM env var")
	flag.StringVar(&wif, "wif", "", "required: private key, alternate: $WIF env var")
	flag.StringVar(&nodeos, "url", "", "required: nodeos api url, alternate: $URL env var")
	flag.StringVar(&sTarget, "target", "2.0", "optional: target price of regaddress in USDC, alternate: $TARGET env var")
	flag.StringVar(&customFees, "fees", "", "optional: JSON file for overriding default fee votes, alternate: $JSON env var")
	flag.IntVar(&frequency, "frequency", 2, "optional: hours to wait between runs (does not apply to AWS Lambda)")
	flag.BoolVar(&once, "x", false, "optional: exit after running once (does not apply to AWS Lambda,) use for running from cron")
	flag.Parse()

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		once = true
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
		update, err = needsBaseFees(nil, actor, api)
		if err != nil && once {
			return err
		}
	default:
		custom := make([]*fio.FeeValue, 0)
		f, err := os.OpenFile(customFees, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("could not open custom fees")
			return err
		}
		j, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		_ = f.Close()
		err = json.Unmarshal(j, &update)
		if err != nil {
			log.Println("could not parse custom fees")
			return err
		}
		update, err = needsBaseFees(custom, actor, api)
		if err != nil && once {
			return err
		}
	}

	if update != nil {
		log.Println("attempting feevote update")
		api.RefreshFees()
		// may help to compress given the size of the request.
		opt.Compress = fio.CompressionZlib
		_, err := api.SignPushActionsWithOpts([]*eos.Action{fio.NewActionWithPermission("fio.fee", "setfeevote", actor, string(perm),
				fio.SetFeeVote{FeeRatios: update, MaxFee: fio.Tokens(fio.GetMaxFeeByAction("setfeevote")), Actor: actor},
			).ToEos()}, &opt.TxOptions,
		)
		if err != nil {
			log.Println(err)
			log.Println("Could not update base fees, has it been an hour? Continuing anyway")
		} else {
			log.Println("feevote updated")
		}
	}

	setMultiplier := func() error {
		prices, err := getGecko()
		if err != nil {
			return err
		}
		if prices.LastUpdated.Before(time.Now().Add(-1 * time.Hour)) {
			return errors.New("coingecko data was stale, aborting")
		}
		avg, err := prices.GetAvg()
		if err != nil {
			return err
		}

		df, err := getRegFioAddrCost()
		if err != nil {
			return err
		}
		defaultFee := float64(df) / 1_000_000_000.0
		multiplier := target / (defaultFee * avg)

		current, err := GetCurMult(actor, api)
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
			_, err := api.SignPushActions(act)
			if err != nil {
				log.Println(err)
				// don't bail, try the ComputeFees call on the way out
			}
		} else {
			log.Printf("Multiplier has not changed enough to re-submit: existing %f, proposed %f\n", current, multiplier)
			return nil
		}

		// this can fail without consequence, try to call it several times across multiple blocks.
		for i := 0; i < 3; i++ {
			_, err = api.SignPushActions(fio.NewActionWithPermission("fio.fee", "computefees", actor, string(perm), fio.ComputeFees{}))
			if err != nil {
				log.Println("Compute fees failed (can safely ignore): ", err.Error())
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		return nil
	}

	err = setMultiplier()
	// don't loop if running as lambda function
	if once {
		return err
	}
	rand.Seed(time.Now().UnixNano()) // #nosec
	ticker := time.NewTicker(time.Duration(frequency) * time.Hour)
	for {
		select {
		case <-ticker.C:
			go func() {
				// add some variability to when this starts, less predictability makes it less likely to be subjected
				// to timing / flash attacks.
				time.Sleep(time.Duration(rand.Intn(300)+1) * time.Second) // #nosec
				if err = setMultiplier(); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

func defaultFee() []*fio.FeeValue {
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
		{EndPoint: "submit_fee_multiplier", Value: 10000000}, // I'm overriding the default here, this should be cheap.
		{EndPoint: "submit_fee_ratios", Value: 70000000},
		{EndPoint: "transfer_fio_address", Value: 60000000},
		{EndPoint: "transfer_fio_domain", Value: 100000000},
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
	sort.Slice(existing, func(i, j int) bool {
		return fees[i].EndPoint < fees[j].EndPoint
	})
	if len(existing) != len(fees) {
		log.Printf("different number of fee values on-chain (%d) vs desired (%d)", len(existing), len(fees))
		return fees, nil
	}
	for i := range existing {
		var sendDefault bool
		if existing[i].EndPoint != fees[i].EndPoint || existing[i].Value != fees[i].Value {
			log.Println("on-chain data differs for desired fee endpoint:", existing[i].EndPoint)
			sendDefault = true
		}
		if sendDefault {
			return fees, nil
		}
	}
	return nil, errors.New("unknown error checking feevote")
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
