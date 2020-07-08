package airdrop

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"os"
	"sync"
	"time"
)

func SendAllTokens(recips []*Recipient, acc *fio.Account, api *fio.API, abort chan interface{}) (report string, errs []error) {
	done := make(chan interface{})
	defer func() {
		close(done)
	}()
	header := `"account","pubkey","amount","successful","txid","confirmed","block_num"` + "\n"

	// this routine will catch the abort channel being closed (triggered by sigint etc in main.go) and dump
	// the current state immediately to stdout, this way if the run is interrupted, it is easier to tell who
	// was paid or not.
	rMux := sync.Mutex{}
	go func() {
		for {
			select {
			case <-done:
				return
			case <-abort:
				rMux.Lock()
				defer rMux.Unlock()
				fmt.Print("\n****** airdrop interrupted before completion, dumping state ******\n\n")
				fmt.Print(header)
				for _, recip := range recips {
					fmt.Printf(
						`"%s","%s","%f","%v","%s","%v","%d"`+"\n",
						recip.Account,
						recip.PubKey,
						float64(recip.Amount)/1_000_000_000.0,
						recip.Success,
						recip.TxId,
						recip.Confirmed,
						recip.BlockNum,
					)
				}
				os.Exit(1)
			}
		}
	}()

	fmt.Println()
	log.Println("sending transactions")
	buf := bytes.NewBufferString(header)
	var success, fail, retry int
	for i := 0; i < maxRetries; i++ {
		allSent := true
		if i > 0 {
			log.Println("Failed transactions detected, retry #", i)
		}
		for x, recip := range recips {
			if recip.Success {
				continue
			}
			// progressively slowdown if we are doing re-sends, may be getting rate limited so chill a little
			if i > 0 {
				time.Sleep(time.Duration(i*500) * time.Millisecond)
				retry += 1
			} else if x%100 == 0 {
				log.Printf("sent %d transactions ...\n", x)
			}
			rMux.Lock()
			err := recip.SendTokens(acc, api)
			rMux.Unlock()
			if err != nil {
				fail += 1
				allSent = false
				errs = append(errs, err)
				continue
			}
			success += 1
		}
		if allSent {
			break
		}
	}
	log.Println(fmt.Sprintf("successfully sent tokens to %d/%d accounts. %d failures, %d retries", success, len(recips), fail, retry))
	fmt.Println("")
	log.Println("waiting for finality to verify transactions")

	finalityFailed := func(err error) bool {
		if err != nil {
			log.Println("not able to confirm finality!", err)
			for _, recip := range recips {
				buf.WriteString(fmt.Sprintf(
					`"%s","%s","%f","%v","%s","false",""`,
					recip.Account,
					recip.PubKey,
					float64(recip.Amount)/1_000_000_000.0,
					recip.Success,
					recip.TxId,
				) + "\n")
			}
			return true
		}
		return false
	}

	gi, err := api.GetInfo()
	if finalityFailed(err) {
		return buf.String() + "\n", errs
	}
	now, err := api.GetInfo()
	if finalityFailed(err) {
		return buf.String() + "\n", errs
	}
	for gi.HeadBlockNum > now.LastIrreversibleBlockNum {
		time.Sleep(6 * time.Second)
		now, err = api.GetInfo()
		if finalityFailed(err) {
			return buf.String() + "\n", errs
		}
		log.Printf("%d blocks until finality...", int64(gi.HeadBlockNum)-int64(now.LastIrreversibleBlockNum))
	}

	fmt.Println("")
	log.Println("verifying transactions")
	confirmed := 0
	for x, recip := range recips {
		if x%100 == 0 {
			log.Printf("verified %d transactions ...\n", x)
		}
		rMux.Lock()
		err = recip.CheckFinal(api)
		rMux.Unlock()
		if err != nil {
			errs = append(errs, err)
		}
		buf.WriteString(fmt.Sprintf(
			`"%s","%s","%f","%v","%s","%v","%d"`,
			recip.Account,
			recip.PubKey,
			float64(recip.Amount)/1_000_000_000.0,
			recip.Success,
			recip.TxId,
			recip.Confirmed,
			recip.BlockNum,
		) + "\n")
		if recip.Confirmed == true {
			confirmed += 1
		}
	}
	log.Printf("confirmed %d/%d transactions are finalized\n", confirmed, len(recips))
	fmt.Println("")
	return buf.String() + "\n", errs
}

func (ar *Recipient) SendTokens(acc *fio.Account, api *fio.API) error {
	if ar.Attempt > maxRetries {
		return errors.New(fmt.Sprintf("too many retries (%d) for %s", ar.Attempt, ar.PubKey))
	}
	resp, err := api.SignPushActions(fio.NewTransferTokensPubKey(acc.Actor, ar.PubKey, ar.Amount))
	if err == nil && resp != nil && resp.TransactionID != "" {
		ar.TxId = resp.TransactionID
		ar.Success = true
		return nil
	}
	ar.Attempt += 1
	return errors.New(fmt.Sprintf("sending to %s failed: %v", ar.PubKey, err))
}

func (ar *Recipient) CheckFinal(api *fio.API) error {
	if ar.TxId == "" {
		return errors.New(ar.PubKey + " has an empty txid, cannot confirm finality")
	}
	t, err := hex.DecodeString(ar.TxId)
	if err != nil {
		return errors.New(ar.PubKey + ": could not decode txid to bytes. " + err.Error())
	}
	tr, err := api.GetTransaction(t)
	if err != nil {
		return errors.New(ar.PubKey + ": could not query txid. " + err.Error())
	}
	if tr != nil && tr.BlockNum > 0 {
		ar.BlockNum = tr.BlockNum
		ar.Confirmed = true
	}
	return nil
}
