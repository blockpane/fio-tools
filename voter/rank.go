package voter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

const (
	thirty uint32 = 30 * 24 * 60 * 120
	one    uint32 = 24 * 60 * 120
)

func RankProducers(eligible []string, cpuRank map[string]int, api *fio.API) ([]string, error) {
	if V {
		log.Println("ranking producers ...")
	}
	var err error
	bps := make(map[string]*BpRank)
	_, err = api.GetInfo()
	if err != nil {
		return nil, err
	}
	for _, bp := range eligible {
		if pa, ok, _ := api.PubAddressLookup(fio.Address(bp), "FIO", "FIO"); ok {
			bps[bp] = &BpRank{Address: fio.Address(bp), CpuScore: cpuRank[bp]}
			bps[bp].bpPubKey = pa.PublicAddress
			bps[bp].Account, err = fio.ActorFromPub(pa.PublicAddress)
			if err != nil {
				continue
			}
			err = bps[bp].getHistory(api)
			if err != nil && V {
				log.Println(err)
			}
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(bps))
	for _, b := range bps {
		// getting bp.json is slow, do it concurrently
		go func(bp *BpRank) {
			defer wg.Done()
			err = bp.getBpJson(api)
			if err != nil && V {
				log.Println(err)
			}
			bp.score()
		}(b)
	}
	wg.Wait()

	sort.Slice(eligible, func(i, j int) bool {
		return bps[eligible[i]].Score > bps[eligible[j]].Score
	})

	// save out a copy of rankings
	func() {
		r := make([]*BpRank, len(eligible))
		for i, bpr := range eligible {
			r[i] = bps[bpr]
		}
		j, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			log.Println(err)
			return
		}
		f, err := os.OpenFile("ranks.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Println(err)
			return
		}
		_, _ = f.Write(j)
		_ = f.Close()
	}()

	return eligible, nil
}

type BpRank struct {
	Address   fio.Address     `json:"address"`
	Score     int             `json:"score"`
	Account   eos.AccountName `json:"account"`
	bpPubKey  string
	bpSignKey string

	FeeVote   int `json:"fee_vote_30d"`
	Msig      int `json:"msig_30d"`
	BpClaim   int `json:"bpclaim_1d"`
	TpidClaim int `json:"tpidclaim_1d"`
	Burn      int `json:"burnexpired_1d"`
	CpuScore  int `json:"cpu_score"`

	// TODO: even more info
	//Monitor      bool `json:"monitor"`
	//ApiAvail     bool `json:"api_avail"`
	//ApiTls       bool `json:"api_tls"`
	//P2pAvail     bool `json:"p2p_avail"`
	//NetApi       bool `json:"net_api"`
	//ProdApi      bool `json:"prod_api"`
	//MissedBlocks int  `json:"missed_blocks"`

	DiffSignKey       bool `json:"diff_sign_key"`
	BpJson            bool `json:"bp_json"`
	BpJsonCors        bool `json:"bp_json_cors"`
	RegValidUrl       bool `json:"valid_url"`
	UsingLinkedOrMsig bool `json:"using_linked_auth_or_msig"`
	HasClaimed        bool `json:"has_claimed_30d"`
}

func (bp *BpRank) score() {
	bp.Score = 0
	if bp.bpPubKey != bp.bpSignKey {
		bp.DiffSignKey = true
		bp.Score += 1
	}
	if bp.UsingLinkedOrMsig {
		bp.Score += 3
	}
	for _, good := range []bool{bp.BpJson, bp.BpJsonCors, bp.RegValidUrl, bp.HasClaimed} {
		if good {
			bp.Score += 1
		}
	}
	//  extra rewards for msig, and feevoting
	bp.Score += (3 * bp.Msig) + (2 * bp.FeeVote) + bp.BpClaim + bp.Burn + bp.CpuScore
}

func (bp *BpRank) getBpJson(api *fio.API) error {
	if bp.Account == "" {
		return errors.New("cannot search: bp.Account is empty")
	}
	bpj, err := api.GetBpJson(bp.Account)
	if err != nil {
		return err
	}
	bp.RegValidUrl = true
	if bpj.Nodes != nil && len(bpj.Nodes) > 0 {
		bp.BpJson = true
	}
	if resp, err := http.Get(bpj.BpJsonUrl); err == nil && resp != nil {
		if resp.Header.Get("Access-Control-Allow-Origin") == "*" {
			bp.BpJsonCors = true
		}
	}
	return nil
}

func (bp *BpRank) getHistory(api *fio.API) error {
	_, bpc, err := GetProducerCompact(bp.Account, api)
	if err != nil || bpc == nil {
		if V {
			log.Println("GetProducerCompact failed: ", err)
		}
		return err
	}
	bp.bpSignKey = bpc.ProducerPublicKey
	if bpc.LastClaimTime.After(time.Now().Add(-720 * time.Hour)) {
		bp.HasClaimed = true
	}
	highest, err := api.GetMaxActions(bp.Account)
	if err != nil {
		return err
	}
	if highest == 0 {
		return nil
	}
	gi, err := api.GetInfo()
	if err != nil {
		if V {
			log.Println(err)
		}
		return err
	}
	var thirtyDays, oneDay uint32 = 0, 0
	if thirty < gi.HeadBlockNum {
		thirtyDays = gi.HeadBlockNum - thirty
	}
	if one < gi.HeadBlockNum {
		oneDay = gi.HeadBlockNum - one
	}
	//dups := make(map[string]bool)
	for i := int64(highest); i > 0; i -= 100 {
		pos := i - 100
		if pos < 0 {
			pos = 0
		}
		at, err := api.GetActions(eos.GetActionsRequest{
			AccountName: bp.Account,
			Pos:         pos,
			Offset:      100,
		})
		if err != nil {
			return nil
		}
		if at == nil || at.Actions == nil || len(at.Actions) == 0 || at.Actions[len(at.Actions)-1].BlockNum < thirtyDays {
			break
		}
		for i := len(at.Actions) - 1; i >= 0; i-- {
			if at.Actions[i].BlockTime.Before(time.Now().Add(-time.Duration(24*30) * time.Hour)) {
				break
			}
			//if at.Actions[i].Trace.Action == nil || at.Actions[i].Trace.Action.Authorization == nil || dups[at.Actions[i].Trace.TransactionID.String()] {
			if at.Actions[i].Trace.Action == nil || at.Actions[i].Trace.Action.Authorization == nil {
				continue
			}
			var fromBp bool
			for _, auth := range at.Actions[i].Trace.Action.Authorization {
				if auth.Actor == bp.Account {
					fromBp = true
					break
				}
			}
			act := string(at.Actions[i].Trace.Action.Name)
			if !fromBp {
				switch act {
				case "propose", "approve", "exec":
					bp.Msig += 1
					fmt.Println(at.Actions[i].Trace.TransactionID.String(), bp.Address)
					//dups[at.Actions[i].Trace.TransactionID.String()] = true
				}
				continue
			}
			switch act {
			case "bundlevote", "setfeemult", "setfeevote":
				bp.FeeVote += 1
				//dups[at.Actions[i].Trace.TransactionID.String()] = true
			case "bpclaim":
				if at.Actions[i].BlockNum >= oneDay {
					bp.BpClaim += 1
					//dups[at.Actions[i].Trace.TransactionID.String()] = true
				}
			case "tpidclaim":
				if at.Actions[i].BlockNum >= oneDay {
					bp.TpidClaim += 1
					//dups[at.Actions[i].Trace.TransactionID.String()] = true
				}
			case "burnexpired":
				if at.Actions[i].BlockNum >= oneDay {
					bp.Burn += 1
					//dups[at.Actions[i].Trace.TransactionID.String()] = true
				}
			case "propose", "approve", "exec":
				bp.Msig += 1
				fmt.Println(at.Actions[i].Trace.TransactionID.String(), bp.Address)
				//dups[at.Actions[i].Trace.TransactionID.String()] = true
			}
			for _, auth := range at.Actions[i].Trace.Action.Authorization {
				if bp.Account == auth.Actor && string(auth.Permission) != "active" {
					bp.UsingLinkedOrMsig = true
				}
			}
		}
	}
	return nil
}

// CpuRanking penalizes for high numbers, averages over 4k get negative score, increasing by 1 per 1,000µs
func CpuRanking(api *fio.API) (map[string]int, error) {
	gi, err := api.GetInfo()
	if err != nil {
		return nil, err
	}
	//through := gi.HeadBlockNum - uint32(60 * 60 * 2) // get CPU stats for last hour
	through := gi.HeadBlockNum - uint32(60*60*2*2) // get CPU stats for last hour

	type prods struct {
		Owner      string `json:"owner"`
		FioAddress string `json:"fio_address"`
	}
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:  "eosio",
		Scope: "eosio",
		Table: "producers",
		Limit: 500,
		JSON:  true,
	})
	if err != nil {
		return nil, err
	}
	if len(gtr.Rows) == 0 {
		fmt.Print("can't map producers, giving up")
		return nil, err
	}
	ps := make([]prods, 0)
	err = json.Unmarshal(gtr.Rows, &ps)
	if err != nil {
		return nil, err
	}
	prodTable := make(map[string]string)
	for _, producer := range ps {
		prodTable[producer.Owner] = producer.FioAddress
	}

	counts := make(map[string][]uint32)
	for i := gi.HeadBlockNum; i >= through; i-- {
		gbt, err := api.HistGetBlockTxids(i)
		if err != nil || gbt == nil || len(gbt.Ids) == 0 {
			continue
		}
		time.Sleep(10 * time.Millisecond)
		gb, err := api.GetBlockByNum(i)
		if err != nil {
			log.Println(err)
			continue
		}
		if gb.Transactions == nil || len(gb.Transactions) == 0 {
			continue
		}
		if counts[string(gb.Producer)] == nil {
			counts[string(gb.Producer)] = make([]uint32, 0)
		}
		for _, tx := range gb.Transactions {
			counts[string(gb.Producer)] = append(counts[string(gb.Producer)], tx.CPUUsageMicroSeconds)
			fmt.Printf("%d  %s  %d\n", i, string(gb.Producer), tx.CPUUsageMicroSeconds)
		}
	}

	averages := make(map[string]uint64)
	sorted := make([]string, 0)
	for k, v := range counts {
		var total uint64
		for _, micro := range v {
			total += uint64(micro)
		}
		averages[k] = total / uint64(len(v))
		sorted = append(sorted, k)
	}

	if V {
		sort.Slice(sorted, func(i, j int) bool {
			return averages[sorted[i]] > averages[sorted[j]]
		})
		p := message.NewPrinter(language.AmericanEnglish)
		p.Printf("average CPU µs by producer, blocks %d through %d:\n", through, gi.HeadBlockNum)
		for _, bp := range sorted {
			p.Printf("    %-25s  (%s) %20d µs\n", prodTable[bp], bp, averages[bp])
		}
	}
	scores := make(map[string]int)
	for bp, avg := range averages {
		scores[prodTable[bp]] = 0
		if avg > 5_000 {
			scores[prodTable[bp]] -= 3 * (int(avg-5000) / 1000)
		}
	}
	return scores, nil
}
