package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

/*

pizza-oracle is technically not an oracle, but I'm calling it that anyway.

This watches for FIO requests and announces them into a discord channel.
Used to support the 2021 Ethdenver FIO Pizza giveaway.

*/

type State struct {
	Highest int `json:"highest"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	nodeos := os.Getenv("URL")
	wif := os.Getenv("WIF")
	webhook := os.Getenv("DISCORD")
	state := os.Getenv("STATE")

	if nodeos == "" || wif == "" || webhook == "" {
		log.Fatal("DISCORD, URL and WIF env variables must be set")
	}
	if state == "" {
		state = "state.json"
	}

	account, api, _, err := fio.NewWifConnect(wif, nodeos)
	if err != nil {
		log.Fatal(err)
	}

	// don't alert on outstanding requests, get highest request ID first ....
	highest := State{}
	func() {
		var f *os.File
		f, err = os.Open(state)
		if err != nil {
			log.Println("starting with default state, may duplicate alerts!", err)
			return
		}
		defer f.Close()
		b := make([]byte, 0)
		b, err = ioutil.ReadAll(f)
		if err != nil {
			log.Println("starting with default state, may duplicate alerts!", err)
			return
		}
		err = json.Unmarshal(b, &highest)
		if err != nil {
			log.Println("starting with default state, may duplicate alerts!", err)
		}
	}()

	// save whenever we update....
	save := func() {
		j, e := json.Marshal(highest)
		if e != nil {
			log.Println(e)
			return
		}
		f, e := os.OpenFile(state, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if e != nil {
			log.Println(e)
			return
		}
		defer f.Close()
		_, _ = f.Write(j)
	}

	reqs := fio.PendingFioRequestsResponse{}
	has := false
	poll := time.NewTicker(time.Minute)

	hit := false
	for {
		<-poll.C
		log.Println("checking for new requests")

		msgBuf := bytes.NewBuffer(nil)
		for i := 0; i < 100; i++ {
			reqs, has, err = api.GetPendingFioRequests(account.PubKey, 100, i*100)
			if err != nil {
				log.Println(err)
				continue
			}
			if !has {
				continue
			}
			if len(reqs.Requests) == 0 {
				continue
			}

			for i := range reqs.Requests {
				if reqs.Requests[i].FioRequestId <= uint64(highest.Highest) {
					continue
				}
				highest.Highest = int(reqs.Requests[i].FioRequestId)
				result, oops := fio.DecryptContent(account, reqs.Requests[i].PayeeFioPublicKey, reqs.Requests[i].Content, fio.ObtRequestType)
				if oops != nil {
					log.Println(oops)
					_, _ = fmt.Fprintf(msgBuf, "❗️ couldn't decrypt request id %d from %s\n", reqs.Requests[i].FioRequestId, reqs.Requests[i].PayeeFioAddress)
					continue
				}
				hit = true
				_, _ = fmt.Fprintf(msgBuf, "🍕 id: %d `%16s %4s %s/%s` 🍕 memo: %s\n",
					reqs.Requests[i].FioRequestId,
					reqs.Requests[i].PayeeFioAddress,
					result.Request.Amount,
					result.Request.TokenCode,
					result.Request.ChainCode,
					result.Request.Memo,
				)
			}
			if reqs.More == 0 {
				break
			}
		}

		if hit && msgBuf.Len() > 10 {
			save()
			_ = postToDiscord(buildDiscordMessage(msgBuf.String()), webhook)
		}
	}
}

func buildDiscordMessage(msg string) *DiscordMessage {
	fmt.Print(msg)
	return &DiscordMessage{
		Username: "Pizza Pizza!",
		Content:  "",
		Embeds: []DiscordEmbed{{
			Title:       "the BUIDLrs need fuel!",
			Description: msg,
		}},
	}
}

func postToDiscord(message *DiscordMessage, url string) error {
	client := &http.Client{}
	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		log.Println(resp)
		if resp.Body != nil {
			b, _ := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			fmt.Println(string(b))
		}
		log.Println(err)
		return err
	}
	return nil
}

type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	AvatarUrl string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Author      DiscordAuthor `json:"author"`
	Title       string        `json:"title,omitempty"`
	Url         string        `json:"url,omitempty"`
	Description string        `json:"description"`
	Color       uint          `json:"color"`
}

type DiscordAuthor struct {
	Name      string         `json:"name,omitempty"`
	Url       string         `json:"url,omitempty"`
	IconUrl   string         `json:"icon_url,omitempty"`
	Fields    []DiscordField `json:"fields,omitempty"`
	Thumbnail string         `json:"thumbnail"`
	Image     string         `json:"image,omitempty"`
	Footer    DiscordField   `json:"footer,omitempty"`
}

type DiscordField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}
