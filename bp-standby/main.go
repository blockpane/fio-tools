package main

import (
	"flag"
	"github.com/fioprotocol/fio-go"
	eos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"log"
	"os"
	"sort"
	"time"
)

var (
	active bool // tracks if currently top 21
	paused bool // is this instance producing blocks?
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	var err error
	api, acc, nodeLog := opts()

	// make sure producer API is even available
	paused, err = api.IsProducerPaused()
	if err != nil {
		log.Fatal(err)
	}

	// always start paused, some danger in this if this program is repeatedly crashing, which is worse,
	// missing blocks or double-producing blocks and causing many small forks?
	if !paused {
		paused = true
		err = api.ProducerPause()
		if err != nil {
			log.Println(err)
			paused = false
		}
	}

	healthy := make(chan string)              // go routine heartbeat channel
	lastHealthy := make(map[string]time.Time) // tracks last heartbeat for routines, too long w/no heartbeat and we bail
	byOrder := make(chan bool)                // notifications for missing blocks
	byRound := make(chan bool)                // notifications for missing entire rounds, in case missed block detection doesn't work
	restored := make(chan bool)               // notification that other node is signing blocks, and we need to stop
	failing := make(chan error)               // errors from routines, too many and we bail

	// pass around a common block so multiple routines aren't hammering the endpoint.
	block := &eos.BlockResp{}
	go func() {
		info := &eos.InfoResp{}
		b := &eos.BlockResp{}
		var err error // intentional shadow
		for {
			time.Sleep(time.Second)
			info, err = api.GetInfo()
			if err != nil {
				failing <- err
				continue
			}
			b, err = api.GetBlockByNum(info.HeadBlockNum)
			if err != nil {
				failing <- err
				continue
			}
			if b == nil {
				continue
			}
			block = b
			healthy <- "block updates"
		}
	}()

	// use two methods to watch for missed blocks. One is by expecting blocks from our account before or after
	// based on sorting in the schedule. Second is by watching for missed rounds based on reversible block states.
	// finally, if we start seeing duplicate blocks signed by our key, we know the primary is back online

	// if head is not incrementing, and the previous producer is immediately before ours, we are missing blocks.
	var neighbors = &[2]string{}
	go missedByOrder(neighbors, block, acc, byOrder, healthy)

	// checking order may not be reliable if multiple producers are missing rounds, this is a fallback that should find
	// if the producer is missing entire rounds. Not ideal, but beats missing many rounds instead of just one.
	go missedRound(block, api, acc, byRound, healthy, failing)

	// if seeing blocks signed by the bp account then it's back online. In this case, watch the nodeos log file
	go duplicateSig(nodeLog, api, acc, restored, failing)

	var (
		failcount int
		unhealthy bool
	)

	startProducing := func() bool {
		if unhealthy || !active {
			return false
		}
		unhealthy = true
		if paused {
			err = api.ProducerResume()
			if err != nil {
				log.Println("could not resume producer: " + err.Error())
				unhealthy = false // force recheck next interval.
				paused = true
			}
			log.Println("enabled block production")
			paused = false
		}
		return true
	}

	topTick := time.NewTicker(time.Minute)

	for {
		select {
		case <-restored:
			if !paused {
				log.Println("pausing block production")
				err = api.ProducerPause()
				if err != nil {
					log.Println(err)
					continue
				}
				paused = true
				log.Println("successfully paused block production")
			}

		case <-byOrder:
			if startProducing() {
				log.Println(acc, "has missed blocks.")
			}

		case <-byRound:
			if startProducing() {
				log.Println(acc, "has missed a round.")
			}

		case h := <-healthy:
			failcount = 0
			lastHealthy[h] = time.Now()

		case fail := <-failing:
			failcount += 1
			if failcount > 10 {
				// anticipated this is running in a container or under systemd control, and will be restarted.
				log.Fatal("too many failed checks, exiting: " + fail.Error())
			}
			log.Println(fail)

		// track our state, are we a top21, is production currently paused?
		case <-topTick.C:
			active, neighbors, err = isTop21(api, acc)
			if err != nil {
				failing <- err
			}
			paused, err = api.IsProducerPaused()
			if err != nil {
				failing <- err
			}
			// check for dead routines, exit if dead
			for rtn := range lastHealthy {
				if lastHealthy[rtn].Before(time.Now().Add(-10 * time.Minute)) {
					log.Fatalf("FATAL: %s routine hasn't sent a heartbeat for %v, exiting.", rtn, time.Now().Sub(lastHealthy[rtn]))
				}
			}
		}
	}
}

func missedByOrder(neighbors *[2]string, block *eos.BlockResp, bp eos.AccountName, missing chan bool, healthy chan string) {
	var produced, wereNext, busy bool
	var missedCounter, lastBlock uint32
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			if busy {
				continue
			}
			busy = true
			healthy <- "missed blocks"
			if !paused {
				busy = false
				continue
			}
			switch string(block.Producer) {
			case neighbors[0]:
				wereNext = true
				produced = false
			case string(bp):
				wereNext = false
				produced = true
				missedCounter = 0
			case neighbors[1]:
				if !produced && wereNext {
					// missed the entire round!
					log.Printf("%s was scheduled to be next, but did not produce\n", bp)
					missing <- true
					wereNext = false
				}
			}
			if wereNext && block.BlockNum == lastBlock {
				missedCounter += 1
			}
			lastBlock = block.BlockNum
			// this would be 4 missed blocks
			if missedCounter > 2 {
				log.Printf("head block failed to increment for ~4 blocks during the schedule for %s, declaring as missing", bp)
				missing <- true
				wereNext = false
			}
			busy = false
		}
	}
}

func missedRound(block *eos.BlockResp, api *fio.API, bp eos.AccountName, missing chan bool, healthy chan string, failed chan error) {
	var lastScheduleVer, lastScheduleLib uint32
	for {
		time.Sleep(6 * time.Second)
		healthy <- "missed rounds"
		if !paused || block.ScheduleVersion <= lastScheduleVer {
			continue
		}
		lastScheduleVer = block.ScheduleVersion

		bhs, err := api.GetBlockHeaderState(block.BlockNum)
		if err != nil {
			failed <- err
			continue
		}
		// make sure we've had a chance for full round on this schedule before checking
		if bhs.PendingSchedule.ScheduleLibNum != lastScheduleLib {
			gsb := &eos.BlockResp{}
			gsb, err = api.GetBlockByNum(bhs.PendingSchedule.ScheduleLibNum)
			if err != nil {
				failed <- err
				continue
			}
			if time.Now().Before(gsb.SignedBlockHeader.Timestamp.Time.Add(6 * time.Minute)) {
				// not long enough to declare missing....
				continue
			}
			lastScheduleLib = bhs.PendingSchedule.ScheduleLibNum
		}
		if ok, lp := bhs.ProducerToLast(fio.ProducerToLastProduced); ok {
			for _, prod := range lp {
				if string(prod.Producer) == string(bp) && prod.BlockNum < block.BlockNum-(21*6) {
					log.Printf("detected %s has missed a round, last produced on %d, %d blocks ago\n", bp, prod.BlockNum, block.BlockNum-prod.BlockNum)
					missing <- true
				}
			}
		}
	}
}

func duplicateSig(file string, api *fio.API, bp eos.AccountName, restored chan bool, failed chan error) {
	// TODO: use https://github.com/hpcloud/tail to watch log file
}

func isTop21(api *fio.API, prod eos.AccountName) (active bool, neighbors *[2]string, err error) {
	ps, err := api.GetProducerSchedule()
	if err != nil {
		return
	}
	prods := make([]string, len(ps.Active.Producers))
	for i, bps := range ps.Active.Producers {
		if string(bps.AccountName) == string(prod) {
			active = true
		}
		prods[i] = string(bps.AccountName)
	}
	if active {
		sort.Strings(prods)
		for i, p := range prods {
			if p == string(prod) {
				switch i {
				case 0:
					neighbors[0] = prods[len(prods)-1]
					neighbors[1] = prods[i+1]
				case len(prods) - 1:
					neighbors[0] = prods[i-1]
					neighbors[1] = prods[0]
				default:
					neighbors[0] = prods[i-1]
					neighbors[1] = prods[i+1]
				}
			}
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
