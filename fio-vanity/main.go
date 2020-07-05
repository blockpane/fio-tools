package main

/*
   Vanity key generator for FIO, works for account or pubkey

      $ fio-vanity ninja
        ninja5fdxaox,FIO5mGka1tCsbXGXspWgZUCcj2E7CCfANgyRr1ok2T7AMkxspNamz,5JGpwmi8Fib77ACV8utUdBiRpcrHxyaxznaHax4J9xnB6ziiLu1
        rate: 111,625 KPS
        ^Csignal: interrupt

*/

import (
	"flag"
	"fmt"
	eos "github.com/fioprotocol/fio-go/imports/eos-fio"
	ecc "github.com/fioprotocol/fio-go/imports/eos-fio/fecc"
	"github.com/mr-tron/base58"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

var o *Options

func main() {
	o = opts()

	printChan := make(chan string)
	found := func(k *key) {
		printChan <- fmt.Sprintf("%s,%s,%s", k.actor, k.pub, k.priv)
	}

	match := func(k *key) {
		switch o.anywhere {
		case false:
			if o.actor {
				if k.i64 == o.i64 {
					found(k)
					return
				}
				if o.leet {
					for _, m := range o.i64s {
						if k.i64 == m {
							found(k)
							return
						}
					}
				}
			}
			if o.pub {
				if strings.HasPrefix(strings.ToLower(k.pub[4:]), o.word) {
					found(k)
					return
				}
				if o.leet {
					for _, m := range o.words {
						if strings.HasPrefix(strings.ToLower(k.pub[4:]), m) {
							found(k)
							return
						}
					}
				}
			}
		default:
			if o.actor {
				if strings.Contains(k.actor, o.word) {
					found(k)
					return
				}
				if o.leet {
					for _, m := range o.words {
						if strings.Contains(k.actor, m) {
							found(k)
							return
						}
					}
				}
			}
			if o.pub {
				if strings.Contains(strings.ToLower(k.pub[4:]), strings.ToLower(o.word)) {
					found(k)
					return
				}
			}
			if o.leet {
				for _, m := range o.words {
					if strings.Contains(strings.ToLower(k.pub[4:]), m) {
						found(k)
						return
					}
				}
			}
		}
	}

	statsChan := make(chan bool)
	go func() {
		pp := message.NewPrinter(language.AmericanEnglish)
		t := time.NewTicker(time.Minute / 2)
		var counter uint64
		for {
			select {
			case <-statsChan:
				counter += 1
			case p := <-printChan:
				fmt.Println(p)
			case <-t.C:
				pp.Printf("rate: %d KPS\n", counter/30)
				counter = 0
			}
		}
	}()

	keyChan := make(chan *key, 2*o.threads)
	go func() {
		k := &key{}
		for {
			select {
			case k = <-keyChan:
				if k.i64 == 0 {
					continue
				}
				go match(k)
			}
		}
	}()

	for i := 0; i < o.threads; i++ {
		go func() {
			for {
				keyChan <- newRandomAccount()
				statsChan <- true
			}
		}()
	}
	select {}
}

type key struct {
	actor string
	i64   uint64
	pub   string
	priv  string
}

func newRandomAccount() *key {
	priv, _ := ecc.NewRandomPrivateKey()
	k := &key{
		pub:  priv.PublicKey().String(),
		priv: priv.String(),
	}
	k.actor, k.i64 = actorFromPub(k.pub, len(o.word))
	return k
}

// ActorFromPub calculates the FIO Actor (EOS Account) from a public key
func actorFromPub(pubKey string, matchBytes int) (string, uint64) {
	const actorKey = `.12345abcdefghijklmnopqrstuvwxyz`
	decoded, _ := base58.Decode(pubKey[3:])
	var result uint64
	i := 1
	for found := 0; found <= 12; i++ {
		if i > 32 {
			return "", 0
		}
		var n uint64
		if found == 12 {
			n = uint64(decoded[i]) & uint64(0x0f)
		} else {
			n = uint64(decoded[i]) & uint64(0x1f) << uint64(5*(12-found)-1)
		}
		if n == 0 {
			continue
		}
		result = result | n
		found = found + 1
	}
	actor := make([]byte, 13)
	actor[12] = actorKey[result&uint64(0x0f)]
	result = result >> 4
	for i := 1; i <= 12; i++ {
		actor[12-i] = actorKey[result&uint64(0x1f)]
		result = result >> 5
	}
	i64, _ := eos.StringToName(string(actor[:matchBytes]))
	return string(actor[:12]), i64
}

type Options struct {
	anywhere bool
	word     string
	i64      uint64
	actor    bool
	pub      bool
	leet     bool
	threads  int
	words    []string
	i64s     []uint64
}

func opts() *Options {
	o := &Options{}

	flag.BoolVar(&o.anywhere, "a", false, "match anywhere, default only beginning")
	flag.BoolVar(&o.pub, "p", false, "search pubkey")
	flag.BoolVar(&o.actor, "actor", true, "search actor/account name")
	flag.BoolVar(&o.leet, "l", false, "allow 1337 speak substitutions")
	flag.IntVar(&o.threads, "t", 2*runtime.NumCPU(), "workers to generate keys")
	flag.Parse()
	o.word = flag.Arg(0)

	leets := make(map[string]bool)
	subs := map[string]string{
		"b": "8",
		"e": "3",
		"g": "9",
		"h": "4",
		"i": "1",
		"o": "0",
		"s": "5",
	}

	if o.leet {
		sub := func(w string) {
			for k, v := range subs {
				last := strings.Replace(w, k, v, -1)
				leets[last] = true
				for l, x := range subs {
					last := strings.Replace(w, l, x, -1)
					leets[last] = true
				}
			}
		}
		sub(o.word)
		for k := range leets {
			sub(k)
		}
		delete(leets, o.word)
		for k := range leets {
			o.words = append(o.words, k)
		}
		o.i64s = make([]uint64, len(o.words))
		for i := range o.words {
			o.i64s[i], _ = eos.StringToName(o.words[i])
		}
		fmt.Println(o.words)
	}
	sort.Strings(o.words)

	if o.word == "" {
		fmt.Printf("usage: %s <options> <word>\n    use -h to see options.", os.Args[0])
		fmt.Println("\nValid search characters are: 12345abcdefghijklmnopqrstuvwxyz")
		os.Exit(1)
	}
	o.i64, _ = eos.StringToName(o.word)
	return o
}
