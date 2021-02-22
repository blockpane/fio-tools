package bulk

import (
	"encoding/json"
	"errors"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"math"
)

type bundleType uint8

const (
	bundleReject bundleType = iota + 1
	bundleRespond
)

type bundles struct {
	NameHash string `json:"namehash"`
	Bundles  int    `json:"bundleeligiblecountdown"` // no chance of a typo with a name like that? lol.
}

// checkTransfers will check if there are enough tokens to handle sending funds, the amount is whole FIO as a float, and
// xfers is how may accounts getting funds.
func checkTransfers(account eos.AccountName, xfers int, amount float64) (need float64, ok bool, err error) {
	Api.RefreshFees()
	fee := fio.GetMaxFee(fio.FeeTransferTokensPubKey)
	actual, err := Api.GetBalance(account)
	if err != nil {
		return 0, false, err
	}
	if actual < fee*float64(xfers)+amount*float64(xfers) {
		return -(actual - (fee*float64(xfers) + amount*float64(xfers))), false, nil
	}
	return 0, true, nil
}

func checkBundles(requests int, bt bundleType) (ok bool, bundleDeficit int, err error) {
	if !fio.Address(Name).Valid() {
		return false, 0, errors.New("invalid FIO address")
	}
	h := fio.I128Hash(Name)
	gtr, err := Api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "fionames",
		LowerBound: h,
		UpperBound: h,
		Limit:      1,
		KeyType:    "i128",
		Index:      "5",
		JSON:       true,
	})
	if err != nil {
		return false, 0, err
	}
	bun := make([]*bundles, 0)
	err = json.Unmarshal(gtr.Rows, &bun)
	if err != nil {
		return false, 0, err
	}
	if len(bun) == 0 {
		return false, 0, errors.New("could not find account")
	}
	if bun[0].NameHash != h {
		return false, 0, errors.New("address mismatch")
	}
	var rejectMult int
	switch bt {
	case bundleReject:
		rejectMult = 1
	default:
		rejectMult = 2
	}
	if bun[0].Bundles < rejectMult*requests {
		return false, int(math.Round(math.Abs(float64(bun[0].Bundles - (int(bundleReject) * requests))))), nil
	}
	return true, 0, nil
}
