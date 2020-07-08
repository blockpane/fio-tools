package main

import (
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/frameloss/fio-tools/fio-domain-airdrop"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

func main() {
	e := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	p := message.NewPrinter(language.AmericanEnglish)

	acc, api, tokens, f, simulate := airdrop.Setup()
	if f != nil {
		defer f.Close()
	}
	recips, needed, err := airdrop.GetRecips(api, tokens)
	e(err)

	balance, err := api.GetBalance(acc.Actor)
	e(err)
	log.Println(p.Sprintf("Airdrop requires %s%d. Current balance: %s%d", fio.FioSymbol, needed/1_000_000_000, fio.FioSymbol, uint64(balance)))
	if balance < float64(needed/1_000_000_000) && !simulate {
		log.Fatal(acc.Actor, " - has insufficient balance for the airdrop")
	}

	sort.Slice(recips, func(i, j int) bool {
		return recips[i].Amount < recips[j].Amount
	})

	if simulate {
		fmt.Println("--- Simulation would have sent ---")
		for _, r := range recips {
			p.Printf("Send %s%d to %s (%s)\n", fio.FioSymbol, r.Amount/1_000_000_000, r.PubKey, r.Account)
		}
		os.Exit(0)
	}

	// ensure that the status of payment is dumped if the run is interrupted, don't want to get halfway
	// through, and not know who still needs payment.
	abort := make(chan interface{}, 1)
	sigc := make(chan os.Signal, 1)
	signal.Notify(
		sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	go func() {
		<-sigc
		close(abort)
		// shouldn't be needed, but ensures no zombies if SendAllTokens doesn't return
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}()

	report, errs := airdrop.SendAllTokens(recips, acc, api, abort)
	if len(errs) > 0 {
		for _, err = range errs {
			log.Println(err)
		}
	}

	if f != nil {
		_, err = f.WriteString(report)
		if err != nil {
			log.Println("writing to file failed, dumping CSV to stdout:")
			fmt.Println(report)
			log.Fatal(err)
		}
		os.Exit(0)
	}
	fmt.Print("No '-out' option provided for .CSV report, dumping to stdout:\n\n")
	fmt.Println(report)
}
