package airdrop

import (
	"github.com/fioprotocol/fio-go"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"log"
	"os"
	"testing"
)

func TestGetRecips(t *testing.T) {
	api, _, err := fio.NewConnection(nil, os.Getenv("NODEOS"))
	if err != nil {
		t.Fatal(err)
	}
	_, amount, err := GetRecips(api, 50.0)
	if err != nil {
		t.Fatal(err)
	}
	p := message.NewPrinter(language.AmericanEnglish)
	log.Println(p.Sprintf("Need %g FIO\n", float64(amount)/1_000_000_000.0))
}
