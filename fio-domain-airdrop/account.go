package airdrop

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go"
	eos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"log"
	"strconv"
	"time"
)

// these accounts will not get tokens
var doNotDrop = map[string]bool{
	"tw4tjkmo4eyd": true,
}

type Recipient struct {
	Account string
	PubKey  string
	Amount  uint64

	Attempt   int
	Success   bool
	TxId      string
	Confirmed bool
	BlockNum  uint32
}

type domOwner struct {
	Account    string `json:"account"`
	Expiration int64  `json:"expiration"`
}

// GetAccountCounts looks at the fio.address domains table and returns a list of recipients, and the total FIO required
func GetRecips(api *fio.API, tokens float64) ([]*Recipient, uint64, error) {
	owners := make(map[string]int)
	request := eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "domains",
		LowerBound: "0",
		Limit:      100,
		JSON:       true,
	}

	var domainCount, ineligible int
	var err error
	gtr := &eos.GetTableRowsResp{}

	log.Println("mapping accounts owning a domain")
	for i := 0; true; i += 100 {
		request.LowerBound = strconv.Itoa(i)
		gtr, err = api.GetTableRows(request)
		if err != nil {
			return nil, 0, err
		}
		rows := make([]domOwner, 0)
		err = json.Unmarshal(gtr.Rows, &rows)
		if err != nil {
			return nil, 0, err
		}

		for _, r := range rows {
			if doNotDrop[r.Account] || r.Expiration < time.Now().Unix() {
				ineligible += 1
				continue
			}
			owners[r.Account] += 1
			domainCount += 1
		}

		if !gtr.More {
			break
		}
	}
	log.Printf("Indexed fio.address domains table: found %d domains, across %d accounts (excluded %d domains as ineligible.)", domainCount, len(owners), ineligible)
	log.Println("getting public keys...")
	var balanceRequired uint64
	fee := fio.Tokens(fio.GetMaxFee(`transfer_tokens_pub_key`))
	recips := make([]*Recipient, 0)
	for k, v := range owners {
		recip := &Recipient{
			Account: k,
			Amount:  fio.Tokens(tokens * float64(v)),
		}
		err = recip.GetPubKey(api)
		if err != nil {
			return nil, 0, err
		}
		balanceRequired += fio.Tokens(tokens*float64(v)) + fee
		recips = append(recips, recip)
	}
	return recips, balanceRequired, nil
}

type getPubKey struct {
	Clientkey string `json:"clientkey"`
}

// GetPubKey uses the fio.address accountmap table to get the FIO pub key for an account. We do not want to trust
// the get_account API endpoint because accounts with updated permissions may not have a pubkey at all, or the
// pub key may point to the wrong account.
func (ar *Recipient) GetPubKey(api *fio.API) error {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "accountmap",
		LowerBound: ar.Account,
		UpperBound: ar.Account,
		KeyType:    "name",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		return err
	}
	keys := make([]getPubKey, 0)
	err = json.Unmarshal(gtr.Rows, &keys)
	if err != nil {
		return err
	}
	if keys == nil || len(keys) == 0 {
		return errors.New("did not find public key for " + ar.Account)
	}
	na, err := fio.ActorFromPub(keys[0].Clientkey)
	if err != nil {
		return err
	}
	if string(na) != ar.Account {
		return errors.New(fmt.Sprintf("pubkey %s did not match account, wanted %s, got %s", keys[0].Clientkey, ar.Account, na))
	}
	ar.PubKey = keys[0].Clientkey
	return nil
}
