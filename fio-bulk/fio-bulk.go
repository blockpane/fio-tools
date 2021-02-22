package bulk

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func slurp() ([]string, error) {
	requests := make([]string, 0)
	reader := bufio.NewReader(F)
	defer F.Close()
	var e error
	var l []byte
	var pending bool
	for {
		l, _, e = reader.ReadLine()
		if e != nil {
			if e.Error() == "EOF" {
				break
			}
			return requests, e
		}
		var id int
		id, e = strconv.Atoi(strings.TrimSpace(string(l)))
		if e != nil {
			log.Println("could not parse line:", l)
			continue
		}
		pending, e = IsPending(id)
		if e != nil {
			log.Println(e)
			continue
		}
		if pending {
			requests = append(requests, strconv.Itoa(id))
		} else {
			log.Println("have already responded to id", id, "skipping")
		}
	}
	if Verbose {
		fmt.Println(requests)
	}
	return requests, nil
}
