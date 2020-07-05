# fio-vanity

## Vanity key generator

For FIO, works for account or pubkey. CPU only, can generate around 100k keys/per second on an circa 2018 i9 laptop,
slightly better on a desktop computer. Searches for six or more characters can take a *very* long time.

Note: searching for a string anchored at the beginning of the account/actor (the default setting) will be significantly faster
because this method uses integer matching, others (`-a`, `-p`, `-actor=false`) rely on string searches and will take >30% longer.

```
      $ fio-vanity ninja
        ninja5fdxaox,FIO5mGka1tCsbXGXspWgZUCcj2E7CCfANgyRr1ok2T7AMkxspNamz,5JGpwmi8Fib77ACV8utUdBiRpcrHxyaxznaHax4J9xnB6ziiLu1
        rate: 111,625 KPS
        ^Csignal: interrupt


      $ fio-vanity -l -p -actor=false test
        [test t3st te5t t35t]
        af1isj2444wj,FIO6tE5thYWMFYp1D7meeFjMpXArFKXLX5YNXraPNz4qCuuhVWg5C,5KWoqLRTdV19aXQ5eXiL4yHuJaEPdMDKarexQ9YVS2eCxRWaHTg
        2ew4ey5xiee5,FIO5t35TxB6KK8t9gCE9aimcPgJLKBGtUB3tsQGS39x7FL7YcFGJj,5JPRCwVfphgqfqHHBuxbvuZYG3KJy1wgRRheGw9h2rnXjvXLGDi
        ex1gx1yfsau5,FIO7tE5TsUTQVX4jm3VEh3ZLpNH9hiT5LL681j1tSUHgub9EGtbZo,5Jo5E8N2W9A6bFtHi6kq8axKaskcWiMzFBieBBf3zYjoddM2HmC
        q4auyds34ysn,FIO8TestqRF23ksn6AD5Mw6dzSToFavBLWXGX1T75UMRsQ8bFzTSj,5Hwd1Rofif22VPr1B2PMjvoPfcrkM18cBGUemyifVvVAR8UtYbE
        ^C
```

## Options

```
$ fio-vanity -h
Usage of fio-vanity:
  -a	match anywhere, default only beginning
  -actor
    	search actor/account name (default true)
  -l	allow 1337 speak substitutions
  -p	search pubkey
  -t int
    	workers to generate keys (default 2 * vcores)
```

