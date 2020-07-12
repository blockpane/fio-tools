package main

import (
	"errors"
	"flag"
	"github.com/fioprotocol/fio-go"
	eos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"github.com/hpcloud/tail"
	"log"
	"os"
	"regexp"
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
	//if !paused {
	//	paused = true
	//	err = api.ProducerPause()
	//	if err != nil {
	//		log.Println(err)
	//		paused = false
	//	}
	//}

	healthy := make(chan string)              // go routine heartbeat channel
	lastHealthy := make(map[string]time.Time) // tracks last heartbeat for routines, if too long w/no heartbeat then bail
	byOrder := make(chan bool)                // notifications for missing blocks
	byRound := make(chan bool)                // notifications for missing entire rounds, in case missed block detection doesn't work
	restored := make(chan bool)               // notification that other node is signing blocks, and to stop local production
	failing := make(chan error)               // errors from routines, too many errors triggers a restart

	// pass around a common block so multiple routines aren't hammering the endpoint.
	block := &blockNumProd{}
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
			block.BlockNum = b.BlockNum
			block.Producer = string(b.Producer)
			block.ScheduleVersion = b.ScheduleVersion
			healthy <- "block updates"
		}
	}()

	// use two methods to watch for missed blocks. One is by expecting blocks from our account before or after
	// based on sorting in the schedule. Second is by watching for missed rounds based on reversible block states.
	// finally, seeing duplicate blocks signed by this key indicates the primary node is back online

	// if head is not incrementing, and the previous producer is immediately before this bp, it is missing blocks.
	var neighbors = &neighbor{}
	go missedByOrder(neighbors, block, acc, byOrder, healthy)

	// checking order may not be reliable if multiple producers are missing rounds, this is a fallback that should find
	// if this producer is missing entire rounds. Not ideal, but beats missing many rounds instead of just one.
	go missedRound(block, api, acc, byRound, healthy, failing)

	// if seeing blocks signed by this bp account then it's back online, stop producing. In this case, watch the nodeos log file
	go duplicateSig(nodeLog, string(acc), restored, healthy, failing)

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

	active, err = isTop21(neighbors, api, acc)
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

		// track our state, are this bp in the top 21, is production currently paused?
		case <-topTick.C:
			active, err = isTop21(neighbors, api, acc)
			if err != nil {
				failing <- err
			}
			paused, err = api.IsProducerPaused()
			if err != nil {
				failing <- err
			}
			// check for dead routines, exit if dead
			for rtn := range lastHealthy {
				if lastHealthy[rtn].Before(time.Now().Add(-5 * time.Minute)) {
					log.Fatalf("FATAL: %s routine hasn't sent a heartbeat for %v, exiting.", rtn, time.Now().Sub(lastHealthy[rtn]))
				}
			}
		}
	}
}

type neighbor struct {
	Before string
	After string
}

type blockNumProd struct {
	Producer string
	BlockNum uint32
	ScheduleVersion uint32
}

func missedByOrder(neighbors *neighbor, block *blockNumProd, bp eos.AccountName, missing chan bool, healthy chan string) {
	for neighbors.Before == "" || block == nil {
		time.Sleep(5*time.Second)
		log.Println("missed block detection not started, waiting for data")
	}
	log.Println("watching for missed blocks")

	var produced, wereNext, busy bool
	var missedCounter int
	lastBlock := block.BlockNum - 1
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
			switch block.Producer {
			case neighbors.Before:
				wereNext = true
				produced = false
			case string(bp):
				wereNext = false
				produced = true
				missedCounter = 0
			case neighbors.After:
				if !produced && wereNext {
					// missed the entire round!
					log.Printf("%s was scheduled to be next, but did not produce\n", bp)
					missing <- true
					wereNext = false
				}
			}
			if block.BlockNum == lastBlock {
				log.Println("block not incrementing", lastBlock, "last producer", block.Producer)
				if wereNext {
					missedCounter += 1
				}
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

func missedRound(block *blockNumProd, api *fio.API, bp eos.AccountName, missing chan bool, healthy chan string, failed chan error) {
	for block == nil {
		time.Sleep(time.Second)
		log.Println("missed round detection not started, waiting for data")
	}
	log.Println("watching for missed rounds")

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
		// ensure at least a full round on this schedule before checking
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
				if string(prod.Producer) == string(bp) && prod.BlockNum < block.BlockNum-(21*13) {
					log.Printf("detected %s has missed a round, last produced on %d, %d blocks ago\n", bp, prod.BlockNum, block.BlockNum-prod.BlockNum)
					missing <- true
				}
			}
		}
	}
}

// duplicateSig watches the nodeos stdout logs for duplicate blocks by the same producer, these will be rejected
// with a 'Block not applied to head' error. This isn't a 100% guarantee that there aren't two active nodes
// producing blocks with the same key, if a block is empty both producers will create identical blocks. This should
// catch it pretty quick though.
func duplicateSig(file string, bp string, restored chan bool, healthy chan string, failed chan error) {
	t, err := tail.TailFile(file, tail.Config{
		Follow: true,
		ReOpen: true,
		MustExist: true,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: 2,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer t.Cleanup()

	healthTick := time.NewTicker(time.Minute)
	var last time.Time
	re := regexp.MustCompile(`Block not applied to head.*signed by (\w{12})`)

	for {
		select {
		case <- t.Dead():
			failed <- errors.New("log watcher died")

		case line := <-t.Lines:
			if match := re.FindStringSubmatch(line.Text); len(match) > 1 && match[1] == bp {
				log.Println(match[1], "produced a duplicate block")
				restored <- true
			}
			last = line.Time

		case <- healthTick.C:
			if last.After(time.Now().Add(-1*time.Minute)) {
				healthy <- "log watcher"
			}
		}
	}
}

func isTop21(neighbors *neighbor, api *fio.API, prod eos.AccountName) (active bool, err error) {
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
					neighbors.Before = prods[len(prods)-1]
					neighbors.After = prods[i+1]
				case len(prods) - 1:
					neighbors.Before = prods[i-1]
					neighbors.After = prods[0]
				default:
					neighbors.Before = prods[i-1]
					neighbors.After = prods[i+1]
				}
			}
		}
	}
	return active, err
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
	flag.StringVar(&a, "a", "", "producer account to watch for")
	flag.StringVar(&logFile, "f", "/var/log/fio/nodeos.log", "nodeos log file for detecting duplicate blocks")
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
