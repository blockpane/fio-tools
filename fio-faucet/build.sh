#!/bin/bash

GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o faucet main.go

docker build -t faucet .
