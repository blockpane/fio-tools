package main

import (
	bulk "github.com/blockpane/fio-tools/fio-bulk"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LUTC | log.LstdFlags | log.Lshortfile)
	bulk.Options()
	switch true {
	case bulk.Nuke:
		deleted, err := bulk.NukeEmAll()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Rejected %d requests\n", deleted)
		if r, _, _ := bulk.Api.GetPendingFioRequests(bulk.Acc.PubKey, 1000, 0); len(r.Requests) > 0 {
			log.Fatal("there are still pending requests, please try again.")
		}
		os.Exit(0)
	case bulk.Query:
		ok, wrote, err := bulk.DumpRequests()
		log.Printf("wrote %d records to %s\n", wrote, bulk.OutFile)
		if !ok {
			log.Println("WARNING: could not retrieve all records, the table row query may have timed out.")
		}
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	case bulk.Confirm:
		// handle positive responses here.
		success, fails, err := bulk.Respond()
		log.Printf("sucessfully responded to %d requests, %d failed from %s\n", success, fails, bulk.InFile)
		if err != nil {
			log.Fatal(err)
		}

	default:
		// finally, delete requests from the list
		rejected, err := bulk.RejectRequests()
		log.Printf("rejected %d requests from %s\n", rejected, bulk.InFile)
		if err != nil {
			log.Fatal(err)
		}

	}

}
