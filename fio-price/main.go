package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {

	var usd float64
	var short bool
	flag.Float64Var(&usd, "dollars", 0.0, "amount to convert to FIO, optional")
	flag.BoolVar(&short, "short", false, "truncate to 4 digits of precision")
	flag.Parse()

	prices, err := getGecko()
	if err != nil {
		log.Fatal(err)
	}

	fioPrice, err := prices.GetAvg()
	if err != nil {
		log.Fatal(err)
	}

	switch true {
	case usd == 0 && short:
		fmt.Println(strings.TrimRight(fmt.Sprintf("%.4f", fioPrice), "0"))
	case usd == 0:
		fmt.Println(strings.TrimRight(fmt.Sprint(fioPrice), "0"))
	case short:
		fmt.Println(strings.TrimRight(fmt.Sprintf("%.4f", usd/fioPrice), "0"))
	default:
		fmt.Println(strings.TrimRight(fmt.Sprint(usd/fioPrice), "0"))
	}

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