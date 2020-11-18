# fio-faucet

This is a faucet. But in FIO-style ... it responds to a [FIO funds request](https://developers.fioprotocol.io/api/api-spec/reference/new-funds-request/new-funds-request)
from a list of allowed keys. It is intended to run in a container, but should work fine as a command line tool. Will send tokens to *any* valid FIO public key as long as the
request is signed by a key on the allowed list.

## Options

Settings can be specified by a command line flag, or environment variable:

```
  -allow string
        list of authorized pubkeys, comma seperated
  -k string
        key for faucet
  -m uint
        Max amount that can be sent in SUF, 1_000_000_000 = ᵮ1.0 (default 10000000000000)
  -u string
        url for nodeos api (default "http://127.0.0.1:8888")
```

Or ENV:

- URL
- KEY
- ALLOWED
