# fio-domain-airdrop

This is a command line utility that will perform an airdrop of tokens (default 50) to every account owning
a domain. It will start by identifying all domain owners, finding their public key, calculating the total FIO required,
and if there are no issues will start the airdrop.

It should be pretty easy to adapt to other circumstances by modifying the queries in the GetRecips function in accounts.go.

If a transaction fails, it will attempt to send the funds 3 times, slowing the pace each successive round in the case
that rate limiting is the issue. Once all transactions have completed, or attempts exhausted, it will verify that the
transactions are on-chain. If a run is interrupted it will immediately dump a .csv file showing what was sent, the error
handling is pretty robust but this at least allows an easy way to tell what accounts did not get funds.

For an example of the output for a run, please see [the example in this repo.](example-output.txt)

### options

Options can be specified via command line option, or as a environment variable.

At the end of the run, it will write a file specified by the `-out` option in .CSV format of the results. If this is not
provided, it will be written to stdout. This should provide the information needed in the case that the airdrop was interrupted,
or had intermittent issues.

```
Usage of fio-airdrop:
     -amount float
       	amount to send in airdrop, env var: AMOUNT (default 50)
     -dry-run
       	do not send tokens, only show what would be done
     -k string
       	WIF key to use, env var: WIF
     -out string
       	filename for saving CSV of results, default stdout, env var: OUT
     -t string
       	TPID for transactions, env var: TPID
     -u string
       	nodoes URL to connect to, env var: NODEOS
```
