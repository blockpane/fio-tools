# fio-bulk-reject

A simple utility for either dumping all FIO requests in csv format, or rejecting a list of requests from a list of
request IDs.

## options

```
  -h string
    	FIO API endpoint to use (default "https://testnet.fioprotocol.io")
  -in string
    	file containing FIO request IDs to reject, incompatible with -out, invokes reqobt::rejectfndreq
  -k string
    	private key in WIF format, if absent will prompt
  -out string
    	file to dump all outstanding FIO requests into, will be in .CSV format and include decrypted request details
```

## query

Using the `-out` argument will generate a .csv file containing all pending FIO requests. For example:

```csv
"request_id","payer","payer_fio","payee","payee_fio","address","amount","chain","token","memo","hash","url"
"123","me@mydomain","FIO6QtJu52ho38zRP4aZCcgtciLAWQUB3CBgXnmwfFfXi6LvfVYyj","you@yourdomain","FIO5NMm9Vf3NjYFnhoc7yxTCrLW963KPUCzeMGv3SJ6zR3GMez4ub","18eYGo7posG4YyKj3yYw5WtQRtLJoCm1H7","0.001000","BTC","BTC","I need money",""
```

## bulk reject

The `-in` argument will read a file, it expects one request id per line. It will send a fio.reqobt::rejectfndreq action
for each ID. Note: if a request has already been rejected, the system will still accept the transaction. Also important
to note, **this utility will not check if the account is using bundled transactions or if it will be spending tokens for
the transactions**

## private key

If a private key isn't provided via the `-k` argument, it will prompt for the key. It's probably not a good idea to
pass it on the command line since it will both be in the shell's command history and will be viewable by anyone on
the system by inspecting the list of running processes.
