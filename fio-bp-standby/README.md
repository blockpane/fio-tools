# fio-bp-standby

This is a tool that will enable or disable block production on a node. It is intended to activate a standby node quickly,
if a producer is not signing, and deactivate just as quickly if it detects the primary node is producing blocks again.

This utility should be run ON the _standby_ node, it does not require any other connections -- all missed block detection
takes place from on-chain data.

_Note: this won't work correctly on EOS because the producer schedule in FIO is sorted by account, not location._

Obviously this requires the `eosio::producer_api_plugin` to be enabled. Don't expose this API to the internet or it
will be subject to abuse.

## Detection:

### Missed blocks

1. Every second the latest block is pulled, and the current producer is checked. If it is the producer immediately
in the schedule before this block producer, and the head block stops incrementing, it will immediately enable production.
This usually ensures recovery before the round is complete, usually within 6 blocks.
1. The above isn't foolproof, if the previous producer is also missing blocks it will not help. To cover this possibity
every rotation the last-produced time is checked for the producer using the get_block_header_state (reversible block log)
and if no blocks have been signed for more than one rotation (and the current schedule is more than one rotation old) it
declares the producer as missing rounds and enables production.

### Detecting duplicate blocks being signed with the same key

To ensure the node stops producing if the primary node recovers, this tool tails the nodeos log file
(default `/var/log/fio/nodeos.log`) and looks for the error `Block not applied to head`. If the account matches, it
immediately disables block production, normally only allowing one or two duplicate blocks total.

## Options

```
  -a string
    	producer account to watch for
  -f string
    	nodeos log file for detecting duplicate blocks (default "/var/log/fio/nodeos.log")
  -u string
    	nodeos API to connect to (default "http://127.0.0.1:8888")
```
