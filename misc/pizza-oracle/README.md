# pizza-oracle

pizza-oracle is technically not an oracle, but I'm calling it that anyway.

This watches for FIO requests and announces them into a discord channel.
Used to support the 2021 Ethdenver FIO Pizza giveaway.

All settings performed via environment variables:

* `URL      ` nodeos RPC-API address
* `WIF      ` private key for account decrypting requests
* `DISCORD  ` webhook URL for a Discord channel
* `STATE    ` file to store highest request notified (defaults to state.json in CWD)
