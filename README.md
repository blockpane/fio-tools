# fio-tools

These are miscellaneous tools for working with the FIO Protocol fioprotocol.io

These are provided with no warranty, use at your own risk.

## Installation

1. Install Go version 1.14 or later.
   - [Go Downloads page](https://golang.org/dl/)
   - On MacOS, [mac brew](https://brew.sh/) tracks the current releases very closely.
   - Ubuntu <= 18.04 also has `ppa:longsleep/golang-backports` but as of July 20' Ubuntu 20.04 is not supported
1. Fetch and install the programs:

```
$ go get -d github.com/frameloss/fio-tools
$ cd ~/go/src/github.com/frameloss/fio-tools/
$ go get ./...
$ go install -ldflags="-s -w" ./...
```

The binaries should be in $GOPATH/bin/ (usually ~/go/bin)
