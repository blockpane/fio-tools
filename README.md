# fio-tools

These are miscellaneous tools for working with the [FIO Protocol](https://github.com/fioprotocol/) fioprotocol.io

These are provided with no warranty, use at your own risk.

## What's here:

Daemons:

 * [fio-bp-standby](fio-bp-standby/) automated failover for standby block producer nodes when primary is failed
 * [fio-fee-vote](fio-fee-vote/) sets FIO fee votes and multipliers
 * [fio-bp-vote](fio-bp-vote/) daemon for ranking and voting for block producers

Command Line utilities:

 * [fio-bulk-reject](fio-bulk-reject/) Can dump a list of FIO requests in CSV, and reject them in bulk, useful when there are thousands of requests
 * [fio-koinly](fio-koinly/) generates CSV files that can be imported into koinly.io for calculating transaction cost-basis
 * [fio-req](fio-req/) utility for sending, viewing, rejecting, or responding to FIO requests
 * [fio-top](fio-top/) *nix top-like command for watching transaction activity (inspired by Cryptolions' monitor site)
 * [fio-vanity](fio-vanity/) vanity key generator

Miscellaneous (not included in binary releases)

 * [fio-domain-airdrop](misc/fio-domain-airdrop/) Sends FIO to all domain holders
 * [fio-faucet](misc/fio-faucet/) A faucet that responds to FIO requests

## Installation

1. Install Go version 1.14 or later.
   - [Go Downloads page](https://golang.org/dl/)
   - On MacOS, [mac brew](https://brew.sh/) tracks the current releases very closely.
   - Ubuntu <= 18.04 also has `ppa:longsleep/golang-backports` but as of July 20' Ubuntu 20.04 is not supported
1. Fetch and install the programs:

```
$ go get -d github.com/blockpane/fio-tools
$ cd ~/go/src/github.com/blockpane/fio-tools/
$ go get ./...
$ go install -ldflags="-s -w" ./...
```

The binaries should be in $GOPATH/bin/ (usually ~/go/bin)

