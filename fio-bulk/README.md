# fio-bulk-reject

A simple utility for either dumping all FIO requests in csv format, rejecting ALL pending requests, or rejecting a list
of requests from a list of request IDs from a file.

## options

```
  -fio float
    	amount of whole FIO (as a float) to send via 'trnsfiopubky' with each response (only applies if '-record=true')
  -in string
    	file containing FIO request IDs to reject, incompatible with -out, invokes reqobt::rejectfndreq
  -k string
    	private key in WIF format, if absent will prompt
  -memo string
    	memo to send with responses, does not apply to rejected requests: only with '-record'
  -name string
    	FIO name for requests (required)
  -nuke
    	don't print, don't check, nuke all pending requests. Incompatible with -in -out
  -out string
    	file to dump all outstanding FIO requests into, will be in .CSV format and include decrypted request details
  -record
    	true sends a 'recordobt' response, false sends a 'rejectfndreq' only applies with '-in' option
  -u string
    	FIO API endpoint to use (default "https://testnet.fioprotocol.io")
  -unknown
    	allow connecting to unknown chain id
  -y	assume 'yes' to all questions
```

## query

Using the `-out` argument will generate a .csv file containing all pending FIO requests. For example:

```csv
"timestamp_utc","request_id","payer","payer_fio","payee","payee_fio","address","amount","chain","token","memo","hash","url"
"2021-02-05 06:17:09.929349 +0000 UTC",123","me@mydomain","FIO6QtJu52ho38zRP4aZCcgtciLAWQUB3CBgXnmwfFfXi6LvfVYyj","you@yourdomain","FIO5NMm9Vf3NjYFnhoc7yxTCrLW963KPUCzeMGv3SJ6zR3GMez4ub","18eYGo7posG4YyKj3yYw5WtQRtLJoCm1H7","0.001000","BTC","BTC","I need money","",""
```

## bulk reject

The `-in` argument will read a file, it expects one request id per line. It will send a fio.reqobt::rejectfndreq action
for each ID. Note: if a request has already been rejected, the system will still accept the transaction. Also important
to note, 

## private key

If a private key isn't provided via the `-k` argument, it will prompt for the key. It's probably not a good idea to
pass it on the command line since it will both be in the shell's command history and will be viewable by anyone on
the system by inspecting the list of running processes.

## Example:

Send $5 of FIO to a list of requests in pay-tokens.txt, this uses the `fio-price` tool (in this repository) to calculate
the exact number of tokens to send.

```shell
fio-bulk -name pizza@fiotestnet -k 5Jxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx -in pay-tokens.txt \
  -record -fio $(fio-price -dollars 5 -short) -memo="congrats, you're a winner"

```

where the pay-tokens.txt would be a file with FIO request IDs, for example:

```
123
234
345
456
```
