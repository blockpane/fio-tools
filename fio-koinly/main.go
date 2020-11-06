package main

import (
	"flag"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Row struct {
	Order       int
	Date        time.Time
	Payer       string
	Payee       string
	Memo        string
	Credit      float64
	Debit       float64
	Fee         float64
	Transaction string
	Description string

	received, sent, fee string
}

func main() {
	var nodeos, account, outfile string

	flag.StringVar(&nodeos, "u", "https://fio.blockpane.com", "nodeos url")
	flag.StringVar(&account, "account", "", "required: account (actor) to generate report for")
	flag.StringVar(&outfile, "o", "", "optional: output file, defaults to <account>.csv")
	flag.Parse()

	if outfile == "" {
		outfile = account + ".csv"
	}

	api, _, err := fio.NewConnection(nil, nodeos)
	if err != nil {
		log.Fatal(err)
	}
	if !api.HasHistory() {
		log.Fatal("This program requires access to a history node")
	}

		f, err := os.OpenFile(account+".csv", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		order := 1
		rows := make([]*Row, 0)
		highest, err := api.GetMaxActions(eos.AccountName(account))
		if err != nil {
			log.Fatal(err)
		}
		if highest > 10_000 {
			log.Printf("WARNING: %s has %d actions, most history nodes truncate at 10,000\n", account, highest)
		}
		out := &eos.ActionsResp{}
		for i := uint32(0); i < highest; i += 100 {
			out, err = api.GetActions(eos.GetActionsRequest{
				AccountName: eos.AccountName(account),
				Pos:         int64(i),
				Offset:      99,
			})
			if err != nil {
				log.Fatal(err)
			}
			feeFor := ""
			for _, action := range out.Actions {
				r := &Row{}
				var add bool
				switch action.Trace.Action.Name {
				case "trnsfiopubky":
					if action.Trace.Action.Authorization[0].Actor == "fio.token" || action.Trace.Receipt.Receiver != "fio.token" {
						continue
					}
					payee, _ := fio.ActorFromPub(action.Trace.Action.ActionData.Data.(map[string]interface{})["payee_public_key"].(string))
					r = &Row{
						Order:       order,
						Date:        action.BlockTime.Time,
						Payer:       string(action.Trace.Action.Authorization[0].Actor),
						Payee:       string(payee),
						Memo:        "Public Key Transfer",
						Transaction: action.Trace.TransactionID.String(),
					}
					if err != nil {
						log.Printf("error parsing amount for txid %s: %s\n", r.Transaction, err.Error())
					}
					if string(action.Trace.Action.Authorization[0].Actor) == account {
						// outgoing payment
						r.Debit = action.Trace.Action.ActionData.Data.(map[string]interface{})["amount"].(float64) / 1_000_000_000
						r.sent = "FIO"
					} else {
						r.Credit = action.Trace.Action.ActionData.Data.(map[string]interface{})["amount"].(float64) / 1_000_000_000
						r.received = "FIO"
					}
					add = true
					order += 1
					feeFor = "transfer to public key"
				case "transfer":
					if action.Trace.Action.ActionData.Data.(map[string]interface{})["from"] == account && action.Trace.Action.ActionData.Data.(map[string]interface{})["to"] == "fio.treasury" {
						split := strings.Split(action.Trace.Action.ActionData.Data.(map[string]interface{})["quantity"].(string), " ")
						amt, err := strconv.ParseFloat(split[0], 64)
						if err != nil {
							log.Println(err)
							break
						}
						r = &Row{
							Order:       order,
							Date:        action.BlockTime.Time,
							Payer:       action.Trace.Action.ActionData.Data.(map[string]interface{})["from"].(string),
							Payee:       action.Trace.Action.ActionData.Data.(map[string]interface{})["to"].(string),
							Debit:       amt,
							Transaction: action.Trace.TransactionID.String(),
							sent:        "FIO",
						}
						order += 1
						add = true
					}
					if action.Trace.Action.ActionData.Data.(map[string]interface{})["from"] == "fio.treasury" && action.Trace.Action.ActionData.Data.(map[string]interface{})["to"] == account {
						split := strings.Split(action.Trace.Action.ActionData.Data.(map[string]interface{})["quantity"].(string), " ")
						amt, err := strconv.ParseFloat(split[0], 64)
						if err != nil {
							log.Println(err)
							break
						}
						r = &Row{
							Order:       order,
							Date:        action.BlockTime.Time,
							Payer:       action.Trace.Action.ActionData.Data.(map[string]interface{})["from"].(string),
							Payee:       action.Trace.Action.ActionData.Data.(map[string]interface{})["to"].(string),
							Credit:      amt,
							Transaction: action.Trace.TransactionID.String(),
							received:    "FIO",
						}
						feeFor = action.Trace.Action.ActionData.Data.(map[string]interface{})["memo"].(string)
						order += 1
						add = true
					}
				case "deleteauth", "linkauth", "updateauth", "unlinkauth":
					feeFor = "permissions"
				case "regproxy", "unregproxy", "voteproducer", "voteproxy":
					feeFor = "voting "
				case "regproducer", "bundlevote", "setfeemult", "setfeevote":
					feeFor = "block producer"
				case "approve", "cancel", "exec", "invalidate", "propose", "unapprove":
					feeFor = "multisig transaction"
				case "addaddress", "burnaddress", "regaddress", "regdomain", "remaddress", "remalladdr", "renewaddress", "renewdomain", "setdomainpub", "xferaddress", "xferdomain":
					feeFor = "FIO address or domain"
				case "cancelfndreq", "newfundsreq", "recordobt", "rejectfndreq":
					feeFor = "FIO funds requests"
				}
				if add {
					r.Description = string(action.Trace.Action.Name)
					r.Memo = feeFor
					rows = append(rows, r)
				}
			}
		}
		size, _ := f.WriteString("Date,Sent Amount,Sent Currency,Received Amount,Received Currency,Fee Amount,Fee Currency,Net Worth Amount,Net Worth Currency,Label,Description,TxHash\n")
		for r := range rows {
			s, _ := f.WriteString(fmt.Sprintf(`"%s","%f","%s","%f","%s","%f","%s","","","%s","%s","%s"`, rows[r].Date.UTC().Format(time.RFC3339), rows[r].Debit, rows[r].sent, rows[r].Credit, rows[r].received, rows[r].Fee, rows[r].fee, rows[r].Description, rows[r].Memo, rows[r].Transaction) + "\n")
			size += s
		}
		err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Done: wrote %d bytes to %s\n", size, outfile)
}
