# fio-price

Command line tool to get current price of FIO token. 

With no options will simply print the cost in USDT/USDC of a token based
on the average cost of FIO on all exchanges provided by coingecko.

The optional `-dollars` flag will print the number of FIO tokens that adds
up to that dollar amount. The `-short` flag will truncate to 4 digits of precision.

```
$ fio-price
0.0992848

$ fio-price -dollars 5
50.36017597859894

$ fio-price  -dollars 5000 -short
50259.8434

```
