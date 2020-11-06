# koinly-csv

A helper for [Koinly](https://koinly.io): builds a .csv that can be imported to help with tax accounting. Based upon their
[universal format](https://help.koinly.io/en/articles/3662999-how-can-i-import-my-own-custom-csv-file)

Usage:

```
Usage of fio-koinly:
  -account string
    	required: account (actor) to generate report for
  -o string
    	optional: output file, defaults to <account>.csv
  -u string
    	nodeos url (default "https://fio.blockpane.com")
```

Example:

```
$ fio-koinly -account aloha3joooqd
2020/11/05 20:36:21 Done: wrote 53195 bytes to aloha3joooqd.csv
$ head aloha3joooqd.csv
Date,Sent Amount,Sent Currency,Received Amount,Received Currency,Fee Amount,Fee Currency,Net Worth Amount,Net Worth Currency,Label,Description,TxHash
"2020-03-26T15:36:40Z","0.000000","","700.000000","FIO","0.000000","","","","trnsfiopubky","transfer to public key","ce8d89fb6aace1689407921e34844212c17a90f2b800792a7871df2712c4f442"
"2020-03-26T15:36:45Z","40.000000","FIO","0.000000","","0.000000","","","","transfer","FIO address or domain","c2d76e0c51e723b830cee1f856280666252a6f54f579787d6faaf62d31a97216"
"2020-03-26T15:46:46Z","200.000000","FIO","0.000000","","0.000000","","","","transfer","block producer","b6e5b413a7230782431689d95db44f5de63c23d521c8ffeb5efbd4ba8737cb0f"
...
```
