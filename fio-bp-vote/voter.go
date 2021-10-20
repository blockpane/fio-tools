package voter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	Url        string
	Perm       string
	Actor      string
	Key        string
	Allowed    string
	Address    string
	Frequency  int
	NumVotes   int
	Dry        bool
	V          bool
	Vmux       sync.Mutex
	skipMissed bool
	LastVote   string
)

// tracks missed blocks
var (
	MissedAfter time.Time
	Missed      = make(map[string]time.Time) // holds those who missed blocks, expires at 3*Frequency
	MissedBlk   = 720                        // how many blocks since last producing that gets you kicked ....
)

func Vote(cpuRank map[string]int, api *fio.API) error {
	Vmux.Lock()
	skipMissed = true
	defer func() {
		Vmux.Unlock()
		skipMissed = false
	}()
	if cpuRank == nil {
		cpuRank = make(map[string]int)
	}

	gi, err := api.GetInfo()
	if err != nil {
		return err
	}
	if gi.HeadBlockTime.Before(time.Now().Add(-10 * time.Minute)) {
		if V {
			log.Println("aborting vote, chain is > 10 minutes behind on server")
			j, _ := json.MarshalIndent(gi, "", "  ")
			fmt.Println(string(j))
		}
		return errors.New("headblock time is > 10 minutes behind")
	}

	eligible, err := getEligible(api)
	if err != nil {
		return err
	}
	votes := len(eligible)
	if votes > NumVotes {
		votes = NumVotes
	}
	eligible, err = RankProducers(eligible, cpuRank, api)
	if err != nil {
		return err
	}

	// since this is a long-running daemon, fees may have changed since last run, ensure it's fresh
	api.RefreshFees()
	// little bit more work when overriding the permission ... but this is best done via a linkauth
	action := fio.NewActionWithPermission("eosio", "voteproducer",
		eos.AccountName(strings.Split(Perm, "@")[0]),
		strings.Split(Perm, "@")[1],
		fio.VoteProducer{
			Producers:  eligible[:votes],
			FioAddress: Address,
			Actor:      eos.AccountName(Actor),
			MaxFee:     fio.Tokens(fio.GetMaxFee(fio.FeeVoteProducer)),
		},
	)
	if Dry {
		j, err := json.MarshalIndent(action, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println("would have voted for:")
		fmt.Println(string(j))
	}

	cur := eligible[:votes]
	sort.Strings(cur)
	lv := strings.Join(cur, ",")
	if lv == LastVote {
		if V {
			log.Println("no vote changes based on ranking")
		}
		return nil
	}
	LastVote = lv

	resp := &eos.PushTransactionFullResp{}
	if !Dry {
		for i := 0; i < 10; i++ {
			resp, err = api.SignPushActions(action)
			if err == nil {
				break
			}
			log.Println(err)
			time.Sleep(6 * time.Second)
		}
		if V {
			log.Println("voted for ", eligible[:votes])
			j, _ := json.Marshal(resp)
			log.Println(string(j))
		}
	}
	if Dry || (resp != nil && err == nil) {
		MissedAfter = time.Now().Add(12 * time.Minute)
		go func() {
			last, err := os.OpenFile(".last-vote", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				if V {
					log.Println(err)
				}
				return
			}
			defer last.Close()
			_, err = last.Write([]byte(lv))
			if err != nil {
				if V {
					log.Println(err)
				}
			}
		}()
	}
	return err
}

func FindMisses(cpuRank map[string]int, api *fio.API) error {
	if skipMissed {
		if V {
			log.Println("vote in progress, skipping missed block check")
		}
		return nil
	}
	gi, err := api.GetInfo()
	if err != nil {
		return err
	}
	if MissedAfter.After(time.Now()) {
		if V {
			log.Println("skipping missed block check, not been long enough")
		}
		return nil
	}
	gbh, err := api.GetBlockHeaderState(gi.HeadBlockNum)
	if err != nil {
		return err
	}
	// don't vote anyone out if there is a pending schedule:
	if gbh.PendingSchedule != nil && gbh.PendingSchedule.Schedule != nil &&
		gbh.PendingSchedule.Schedule.Producers != nil && len(gbh.PendingSchedule.Schedule.Producers) > 0 {
		if V {
			log.Println("there is a pending schedule update")
		}
		MissedAfter = time.Now().Add(6 * time.Minute)
		return nil
	}

	ok, ptl := gbh.ProducerToLast(fio.ProducerToLastProduced)
	if !ok {
		return errors.New("could not get last produced")
	}
	// before claiming someone is missing blocks, make sure they are in the active schedule:
	active := make([]string, 0)
	gps, err := api.GetProducerSchedule()
	if err == nil && gps.Active.Producers != nil {
		for _, p := range gps.Active.Producers {
			active = append(active, string(p.AccountName))
		}
	}
	var slacker bool
	for _, last := range ptl {
		//if V {
		//	fmt.Printf("%s last produced %d blocks ago\n", last.Producer, gi.HeadBlockNum-last.BlockNum)
		//}
		isactive := false
		if last.BlockNum < gi.HeadBlockNum-uint32(MissedBlk) {
			for _, p := range active {
				if p == string(last.Producer) {
					isactive = true
					break
				}
			}
			if !isactive {
				continue
			}
			log.Println(last.Producer, " is missing blocks.")
			badAddr, pc, err := GetProducerCompact(last.Producer, api)
			if err != nil {
				log.Println(err)
				continue
			}
			Missed[badAddr] = time.Now().Add(time.Duration(3*Frequency) * time.Hour)
			if !strings.Contains(LastVote, pc.FioAddress) {
				log.Println(pc.FioAddress, " is not on our list, skipping")
				continue
			}
			log.Println(pc.FioAddress, " is on our list, recalculating votes")
			slacker = true
		}
	}
	if slacker {
		func() {
			if j, err := json.Marshal(Missed); err == nil {
				f, err := os.OpenFile(".voter-missed", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
				if err != nil {
					if V {
						log.Println(err)
					}
					return
				}
				defer f.Close()
				n, err := f.Write(j)
				if err != nil && V {
					log.Println(err)
					return
				}
				if V {
					log.Printf("%+v\n", Missed)
					log.Println("wrote ", n, "bytes to .voter-missed")
				}
			}
		}()
		return Vote(cpuRank, api)
	}
	return nil
}

func getEligible(api *fio.API) ([]string, error) {
	gp, err := api.GetFioProducers()
	if err != nil {
		return nil, err
	}
	registered := make(map[string]bool)
	for _, p := range gp.Producers {
		if p.IsActive == 0 || p.TotalVotes == `0.00000000000000000` {
			continue
		}

		// don't rank producers with expired addresses
		var reg fio.FioNames
		var found bool
		reg, found, err = api.GetFioNamesForActor(string(p.Owner))
		if !found || err != nil {
			continue
		}
		if !func() bool {
			for _, address := range reg.FioAddresses {
				if string(p.FioAddress) == (address.FioAddress) {
					t, e := time.Parse("2006-01-02T15:04:05", address.Expiration)
					if e != nil {
						log.Println("error parsing expiration date for", p.FioAddress)
						return false
					}
					if t.After(time.Now()) {
						return true
					}
					log.Println(p.FioAddress, "is expired!")
				}
			}
			return false
		}() {
			continue
		}

		registered[string(p.FioAddress)] = true
	}
	if V {
		log.Println(len(registered), " producers are marked as active")
	}

	eligible := make([]string, 0)
	prods := make([]string, 0)
	if Allowed != "" {
		f, e := os.Open(Allowed)
		if e != nil {
			if V {
				log.Println(e)
			}
			return nil, e
		}
		defer f.Close()
		fb, e := ioutil.ReadAll(f)
		if e != nil {
			if V {
				log.Println(e)
			}
			return nil, e
		}
		prods = strings.Split(string(fb), "\n")
		rand.Seed(time.Now().UnixNano())
	} else {
		for p := range registered {
			prods = append(prods, p)
		}
	}

	// randomize order
	sort.Slice(prods, func(int, int) bool {
		return rand.Intn(10)%2 == 0
	})
	for _, prospect := range prods {
		prospect = strings.TrimSpace(prospect)
		switch false {
		case !fio.Address(prospect).Valid() || !strings.HasPrefix(prospect, "#"):
			log.Println(prospect + " is not a valid fio address")
		case registered[prospect]:
			// nop, inactive in producers table
		default:
			func() {
				if time.Now().Before(Missed[prospect]) {
					if V {
						log.Println(prospect, " not considered, they are missing blocks")
					}
					return
				}
				eligible = append(eligible, prospect)
			}()
		}

	}
	if len(eligible) == 0 {
		return nil, errors.New("no eligible producers")
	}
	return eligible, nil
}

// ProducerCompact trims the response to only what we need.
type ProducerCompact struct {
	FioAddress        string       `json:"fio_address"`
	TotalVotes        string       `json:"total_votes"`
	Url               string       `json:"url"`
	LastClaimTime     eos.JSONTime `json:"last_claim_time"`
	LastBpClaim       int64        `json:"last_bpclaim"`
	ProducerPublicKey string       `json:"producer_public_key"`
}

func GetProducerCompact(acc eos.AccountName, api *fio.API) (addr string, pc *ProducerCompact, err error) {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: string(acc),
		UpperBound: string(acc),
		KeyType:    "name",
		Index:      "4",
		JSON:       true,
	})
	if err != nil {
		return "", nil, err
	}
	results := make([]*ProducerCompact, 0)
	err = json.Unmarshal(gtr.Rows, &results)
	if err != nil {
		return "", nil, err
	}
	if len(results) != 1 || results[0] == nil {
		return "", nil, errors.New("invalid query result")
	}
	if !fio.Address(results[0].FioAddress).Valid() {
		return "", nil, errors.New("invalid address")
	}
	return results[0].FioAddress, results[0], nil
}
