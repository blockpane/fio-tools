#!/bin/bash
set -vx

rm -fr dist/*
mkdir -p dist/fio-tools-win
mkdir -p dist/fio-tools-darwin
mkdir -p dist/fio-tools-linux

VER=$(git describe --tags --always --long)

while read line; do
  n=$(echo $line |awk -F/ '{print $1}')
  GOOS=linux go build -ldflags="-s -w" -o dist/fio-tools-linux/${n} $line/main.go
  CGO_LDFLAGS="-mmacosx-version-min=10.14" CGO_CFLAGS="-mmacosx-version-min=10.14" GOOS=darwin go build -ldflags="-s -w" -o dist/fio-tools-darwin/${n} $line/main.go
  if [[ "${n}" != "fioreq" ]]; then
    GOOS=windows go build -ldflags="-s -w" -o dist/fio-tools-win/${n} $line/main.go
  fi
done << EOF
bp-standby
fioreq
fio-bulk-reject
bp-standby
fio-faucet
voter/cmd
fio-koinly
fio-vanity
fiotop
EOF

pushd dist
ls -1 |while read line; do
  zip "${line}-${VER}" "${line}"
  rm -fr "${line}"
done
popd

