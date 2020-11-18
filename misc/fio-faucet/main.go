package main

import (
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	var (
		url         string
		max         uint64
		faucet      string
		allow       string
		allowedList []string
	)

	flag.StringVar(&url, "u", "http://127.0.0.1:8888", "url for nodeos api")
	flag.StringVar(&allow, "allow", "", "list of authorized pubkeys, comma seperated")
	flag.StringVar(&faucet, "k", "", "key for faucet")
	flag.Uint64Var(&max, "m", 10000000000000, "Max amount that can be sent in SUF, 1_000_000_000 = ᵮ1.0")
	flag.Parse()

	envUrl := os.Getenv("URL")
	if envUrl != "" {
		url = envUrl
	}
	envFaucet := os.Getenv("KEY")
	if envFaucet != "" {
		faucet = envFaucet
	}

	envAllowlist := os.Getenv("ALLOWED")
	if envAllowlist != "" {
		allow = envAllowlist
	}
	allowedList = strings.Split(allow, ",")
	if len(allowedList) == 0 {
		log.Fatal("no allowed-pubkeys list provided.")
	}
	for i := range allowedList {
		allowedList[i] = strings.TrimSpace(allowedList[i])
	}

	t := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-t.C:
			account, err := fio.NewAccountFromWif(faucet)
			if err != nil {
				log.Fatal(err)
			}
			api, opts, err := fio.NewConnection(account.KeyBag, url)
			if err != nil {
				log.Println(err)
				continue
			}
			_, _, _ = account.GetNames(api)

			pending, hasPending, err := api.GetPendingFioRequests(account.PubKey, 100, 0)
			if err != nil {
				log.Println(err)
				continue
			}
			if !hasPending {
				continue
			}

			rejectRequests := make([]uint64, 0)
			for _, request := range pending.Requests {

				// Get payee pub key, check if on allowed list
				payeePub, found, err := api.PubAddressLookup(fio.Address(request.PayeeFioAddress), "FIO", "FIO")
				if err != nil {
					rejectRequests = append(rejectRequests, request.FioRequestId)
					log.Println(err)
					continue
				}
				if !found {
					rejectRequests = append(rejectRequests, request.FioRequestId)
					continue
				}
				isAllowed := false
				for _, allowed := range allowedList {
					if allowed == payeePub.PublicAddress {
						isAllowed = true
						break
					}
				}
				if !isAllowed {
					log.Printf("Rejecting request from %s -- not on list of authorized accounts\n", request.PayeeFioAddress)
					rejectRequests = append(rejectRequests, request.FioRequestId)
					continue
				}

				// decrypt the request, ensure we can (or are willing) to cover the amount
				decrypted, err := fio.DecryptContent(account, payeePub.PublicAddress, request.Content, fio.ObtRequestType)
				if err != nil {
					log.Println(request.PayeeFioAddress + ": error decrypting content")
					rejectRequests = append(rejectRequests, request.FioRequestId)
					log.Println(err)
					continue
				}
				fmt.Printf("Processing request:\n%#v\n", decrypted.Request)
				if decrypted.Request.ChainCode != "FIO" || decrypted.Request.TokenCode != "FIO" {
					log.Println(request.PayeeFioAddress + ": did not request FIO tokens")
					rejectRequests = append(rejectRequests, request.FioRequestId)
					continue
				}
				f, err := strconv.ParseFloat(decrypted.Request.Amount, 64)
				if err != nil {
					log.Println(request.PayeeFioAddress + ": couldn't parse amount")
					rejectRequests = append(rejectRequests, request.FioRequestId)
					log.Println(err)
					continue
				}
				if fio.Tokens(f) > max {
					log.Println(request.PayeeFioAddress + ": too greedy, rejecting.")
					rejectRequests = append(rejectRequests, request.FioRequestId)
					continue
				}
				myBalance, err := api.GetCurrencyBalance(account.Actor, "FIO", "fio.token")
				if err != nil {
					log.Println(err.Error() + " continuing anyway ...")
				}
				if uint64(myBalance[0].Amount) < fio.Tokens(f+fio.GetMaxFee(fio.FeeTransferTokensPubKey)+fio.GetMaxFee(fio.FeeRecordObtData)) {
					log.Println(request.PayeeFioAddress + ": Insufficient balance!")
					rejectRequests = append(rejectRequests, request.FioRequestId)
					continue
				}

				// send the funds:
				txResult, err := api.SignPushTransaction(
					fio.NewTransaction(
						[]*fio.Action{fio.NewTransferTokensPubKey(account.Actor, decrypted.Request.PayeePublicAddress, fio.Tokens(f))},
						opts,
					),
					opts.ChainID, fio.CompressionNone,
				)
				if err != nil {
					log.Println(request.PayeeFioAddress + ": transfer failed: " + err.Error())
					rejectRequests = append(rejectRequests, request.FioRequestId)
					log.Println(err)
					continue
				}
				log.Printf("Sent %s %f to %s with txid %s\n", fio.FioSymbol, f, decrypted.Request.PayeePublicAddress, txResult.TransactionID)

				// record that it has been sent, which will remove it from our pending:
				rec := fio.ObtRecordContent{
					PayerPublicAddress: account.PubKey,
					PayeePublicAddress: decrypted.Request.PayeePublicAddress,
					Amount:             fmt.Sprintf("%f", f),
					ChainCode:          "FIO",
					TokenCode:          "FIO",
					Hash:               txResult.TransactionID,
				}
				content, _ := rec.Encrypt(account, payeePub.PublicAddress)
				_, err = api.SignPushTransaction(fio.NewTransaction(
					[]*fio.Action{fio.NewRecordSend(account.Actor, fmt.Sprintf("%d", request.FioRequestId), account.Addresses[0].FioAddress, request.PayeeFioAddress, content)},
					opts,
				),
					opts.ChainID, fio.CompressionNone,
				)
				if err != nil {
					rejectRequests = append(rejectRequests, request.FioRequestId)
					log.Println(request.PayeeFioAddress + ": " + err.Error())
					continue
				}
			}
			for _, rejectId := range rejectRequests {
				_, err := api.SignPushTransaction(fio.NewTransaction(
					[]*fio.Action{fio.NewRejectFndReq(account.Actor, fmt.Sprintf("%d", rejectId))},
					opts,
				), opts.ChainID, fio.CompressionNone)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}
