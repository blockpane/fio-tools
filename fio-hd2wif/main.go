package main

import (
	"bufio"
	"flag"
	"fmt"
	fiox "github.com/blockpane/fio-extras"
	"io/ioutil"
	"os"
	"regexp"
)

func main() {
	var index, newHdWords int
	var newHd bool
	var inFile string
	var err error

	flag.BoolVar(&newHd, "n", false, "Generate a new HD phrase and print the first WIF")
	flag.IntVar(&newHdWords, "w", 24, "number of words for new nmemonic, valid values are: 12, 15, 18, 21, or 24")

	flag.IntVar(&index, "i", 0, "optional: which key index to derive, default 0")
	flag.StringVar(&inFile, "f", "", "optional: Read HD mnemonic from file")
	flag.Parse()

	if newHd {
		hd, err := fiox.NewRandomHd(newHdWords)
		if err != nil {
			panic(err)
		}
		k, err := hd.KeyAt(index)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s\n%s\n(index %d)\n\n", hd.String(), k.Keys[0].String(), index)
		return
	}

	var nmemonic string
	switch inFile {
	case "":
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Please enter the mnemonic phrase: ")
		nmemonic, err = reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
	default:
		f := &os.File{}
		f, err = os.OpenFile(inFile, os.O_RDONLY, 0600)
		if err != nil {
			panic(err)
		}
		kb := make([]byte, 0)
		kb, err = ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}
		_ = f.Close()
		nmemonic = string(kb)
	}

	rex := regexp.MustCompile(`[^\w ]`)
	hd, err := fiox.NewHdFromString(rex.ReplaceAllString(nmemonic, ""))
	if err != nil {
		panic(err)
	}
	k, err := hd.KeyAt(index)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n(index %d)\n\n", k.Keys[0].String(), index)

}
