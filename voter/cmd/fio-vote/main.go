package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/blockpane/fio-tools/voter"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	flag.StringVar(&voter.Url, "u", "", "url for connect")
	flag.StringVar(&voter.Perm, "p", "", "permission, if not using 'active'")
	flag.StringVar(&voter.Actor, "a", "", "actor")
	flag.StringVar(&voter.Address, "address", "", "fio address")
	flag.StringVar(&voter.Key, "k", "", "wif key")
	flag.StringVar(&voter.Allowed, "allowed", "", "plaintext file of producers eligible for votes: FIO address, 1 per line")
	flag.IntVar(&voter.Frequency, "h", 24, "how often (hours) to run")
	flag.IntVar(&voter.NumVotes, "n", 30, "how many (max) producers to vote for")
	flag.BoolVar(&voter.Dry, "dry-run", false, "don't push transactions, only print what would have been done.")
	flag.BoolVar(&voter.V, "v", false, "verbose logging")
	flag.Parse()

	switch "" {
	case voter.Url, voter.Actor, voter.Key, voter.Address:
		fmt.Println("invalid options, use '-h' for help.")
		os.Exit(1)
	}
	if voter.Perm == "" {
		voter.Perm = voter.Actor + "@" + "active"
	}
	if voter.Allowed == "" {
		log.Println("no allowed-producers list provided: will consider any block producer for voting. \n***** Are you sure this is what you want? *****")
	} else {
		stat, err := os.Stat(voter.Allowed)
		if err != nil {
			panic(err)
		}
		if stat.Size() == 0 {
			fmt.Println("empty producer list")
			os.Exit(1)
		}
	}

	log.Println("fio-voter starting")

	// best effort to save and reload status
	func() {
		if vs, err := os.Open(".voter-missed"); err == nil && vs != nil {
			defer vs.Close()
			j, err := ioutil.ReadAll(vs)
			if err != nil {
				return
			}
			_ = json.Unmarshal(j, &voter.Missed)
			if voter.V && len(voter.Missed) > 0 {
				log.Println("restored missed blocks map")
			}
		} else if voter.V {
			log.Println(err)
		}
		if lv, err := os.Open(".last-vote"); err == nil && lv != nil {
			defer lv.Close()
			b, err := ioutil.ReadAll(lv)
			if err != nil {
				return
			}
			voter.LastVote = string(b)
		}
	}()

	_, api, _, err := fio.NewWifConnect(voter.Key, voter.Url)
	if err != nil {
		panic(err)
	}
	if !api.HasHistory() {
		log.Println(voter.Url, "does not have v1 history enabled.")
		os.Exit(1)
	}
	voter.MissedAfter = time.Now()
	log.Println("ranking producers CPU performance")
	hourRank, err := voter.CpuRanking(api)
	if err != nil || hourRank == nil {
		hourRank = make(map[string]int)
	}
	cpuRank := make(map[string]int)
	twoDays := make([]map[string]int, 48)

	// save last 48 hours of rankings, use the worst performance as the score
	updateCpuRank := func(m map[string]int) {
		cpuRank = make(map[string]int)
		twoDays = append(twoDays[1:], m)
		for _, hour := range twoDays {
			if hour != nil && len(hour) > 0 {
				for k, v := range hour {
					if cpuRank[k] > v {
						cpuRank[k] = v
					}
				}
			}
		}
		if voter.V {
			fmt.Println(m)
			fmt.Println(cpuRank)
		}
	}
	updateCpuRank(hourRank)

	// allow ticker override.
	if os.Getenv("IMMEDIATE") == "true" {
		err = voter.FindMisses(cpuRank, api)
		if err != nil {
			log.Println(err)
		}
		if voter.MissedAfter.Before(time.Now()) {
			err = voter.Vote(cpuRank, api)
			if err != nil {
				log.Println(err)
			}
		}
	}

	tick := time.NewTicker(time.Duration(voter.Frequency) * time.Hour)
	cpuTick := time.NewTicker(time.Hour)
	mux := sync.Mutex{}
	missedTick := time.NewTicker(time.Minute)

	for {
		select {
		case <-tick.C:
			if voter.V {
				log.Println("starting scheduled vote run")
			}
			mux.Lock()
			err = voter.Vote(cpuRank, api)
			mux.Unlock()
			if err != nil {
				log.Println(err)
				// wait, retry once ...
				time.Sleep(time.Duration((voter.Frequency*60)/10) * time.Minute)
				mux.Lock()
				err = voter.Vote(cpuRank, api)
				mux.Unlock()
				if err != nil {
					log.Println(err)
				}
			}
		case <-cpuTick.C:
			mux.Lock()
			if voter.V {
				log.Println("ranking producers CPU performance")
			}
			hourRank, err := voter.CpuRanking(api)
			if err != nil || hourRank == nil {
				log.Println("invalid cpu rank:", err)
				mux.Unlock()
				continue
			}
			updateCpuRank(hourRank)
			mux.Unlock()
		case <-missedTick.C:
			if voter.V {
				log.Println("searching for missed blocks")
			}
			err = voter.FindMisses(cpuRank, api)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
