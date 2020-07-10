package main

import (
	"flag"
	"github.com/fioprotocol/fio-go"
	eos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"log"
	"os"
	"time"
)

var active bool // tracks if currently top 21

func main() {
	log.SetFlags(log.LstdFlags|log.Lshortfile|log.LUTC)
	var err error
	api, acc, nodeLog := opts()

	// always start paused
	err = api.ProducerPause()
	if err != nil {
		log.Println(err)
	}

	healthy := make(chan bool)
	byOrder := make(chan bool)
	byRound := make(chan bool)
	restored := make(chan bool)
	failing := make(chan error)

	// pass around a common block response so multiple routines aren't hammering the endpoint.
	block := &eos.BlockResp{}
	go func() {
		for {
			time.Sleep(time.Second)
			info := &eos.InfoResp{}
			info, err = api.GetInfo()
			if err != nil {
				log.Println(err)
				continue
			}
			b, err := api.GetBlockByNum(info.HeadBlockNum)
			if err != nil {
				log.Println(err)
				continue
			}
			if b == nil {
				continue
			}
			block = b
		}
	}()

	// use two methods to watch for missed blocks. One is by expecting blocks from our account before or after
	// based on sorting in the schedule. Second is by watching for missed rounds based on reversible block states.
	// finally, if we start seeing duplicate blocks signed by our key, we know the primary is back online

	// if head is not incrementing, and the previous producer is immediately before ours, we are missing blocks.
	go missedByOrder(block, api, acc, byOrder, healthy, failing)

	// checking order may not be reliable if multiple producers are missing rounds, this is a fallback that should find
	// if the producer is missing entire rounds. Not ideal, but beats missing many rounds instead of just one.
	go missedRound(block, api, acc, byRound, healthy, failing)

	// if seeing blocks signed by the bp account then it's back online. In this case, watch the nodeos log file
	go duplicateSig(nodeLog, api, acc, restored, failing)

	var (
		paused bool
		failcount int
		unhealthy bool
	)

	missing := func() bool {
		if unhealthy || !active {
			return false
		}
		unhealthy = true
		if paused {
			err = api.ProducerResume()
			if err != nil {
				log.Println("could not resume producer: "+err.Error())
				unhealthy = false // force recheck next interval.
				paused = true
			}
			log.Println("missing blocks detected, enabling block production")
			paused = false
		}
		return true
	}

	topTick := time.NewTicker(time.Minute)

	for {
		select {
		case <- restored:
			if !paused {
				err = api.ProducerPause()
				if err != nil {
					log.Println(err)
					continue
				}
				paused = true
			}

		case <-byOrder:
			if missing() {
				log.Println(acc, "has missed blocks.")
			}

		case <-byRound:
			if missing() {
				log.Println(acc, "has missed a round.")
			}

		case <-healthy:
			failcount = 0

		case fail := <-failing:
			failcount += 1
			if failcount > 10 {
				// anticipated this is running in a container or under systemd control, and will be restarted.
				log.Fatal("too many failed checks, exiting: "+fail.Error())
			}
			log.Println(fail)

		// track our state, are we a top21, is production currently paused?
		case <-topTick.C:
			active, _, err = isTop21(api, acc)
			if err != nil {
				failing <-err
			}
			paused, err = api.IsProducerPaused()
			if err != nil {
				failing <- err
			}
		}
	}
}

func missedByOrder(block *eos.BlockResp, api *fio.API, bp eos.AccountName, missing chan bool, healthy chan bool, failed chan error) {}

func missedRound(block *eos.BlockResp, api *fio.API, bp eos.AccountName, missing chan bool, healthy chan bool, failed chan error) {
	var lastSchedule uint32
	for {
		time.Sleep(6*time.Second)
		if block.ScheduleVersion <= lastSchedule {
			continue
		}
		lastSchedule = block.ScheduleVersion

		bhs, err := api.GetBlockHeaderState(block.BlockNum)
		if err != nil {
			failed <-err
			continue
		}
		// make sure we've had a full round on this schedule before checking
		if bhs.PendingSchedule.ScheduleLibNum > block.BlockNum - (21 * 12) {
			continue
		}
	}
	// TODO: pull code from voter utility to find missed rounds
}

func duplicateSig(file string, api *fio.API, bp eos.AccountName, restored chan bool, failed chan error) {
	// TODO: use https://github.com/hpcloud/tail to watch log file
}

func isTop21(api *fio.API, prod eos.AccountName) (active bool, schedule uint32, err error) {
	ps, err := api.GetProducerSchedule()
	if err != nil {
		return
	}
	schedule = ps.Active.Version
	for _, bps := range ps.Active.Producers {
		if string(bps.AccountName) == string(prod) {
			active = true
			return
		}
	}
	return
}

func opts() (api *fio.API, account eos.AccountName, logFile string) {
	var err error
	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	var a, url string
	flag.StringVar(&url, "u", "http://127.0.0.1:8888", "nodeos API to connect to")
	flag.StringVar(&a, "-a", "", "producer account to watch for")
	flag.StringVar(&logFile, "-f", "/var/log/fio/nodeos.log", "nodeos log file for detecting duplicate blocks")
	flag.Parse()

	api, _, err = fio.NewConnection(nil, url)
	fatal(err)
	// will error if /v1/producer api is not enabled.
	_, err = api.IsProducerPaused()
	fatal(err)

	if len(a) != 12 {
		log.Fatal("account '-a' should be 12 characters")
	}
	account = eos.AccountName(a)

	f, err := os.OpenFile(logFile, os.O_RDONLY, 0644)
	fatal(err)
	_, err = f.Stat()
	fatal(err)

	return
}
