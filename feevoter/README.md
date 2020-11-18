# feevoter

This is a utility for setting FIO fees. It has the following features:

1. Looks up current prices from CoinGecko, and averages the USDT and USDC trading pairs from all listed exchanges.
1. Sets base fee votes if they differ from requested (see default feevote values below.)
1. Sets fee multiplier to desired cost of regaddress in USD (default $2.00)
1. Can run from cron (use -x), as a daemon (default 2 hour loop), or from AWS Lambda (auto detects if running in Lambda)
1. Will not update prices if the change is small (.15 or less change in multiplier), or if the multiplier changes by more than 25%
1. When running as a daemon, will add a random delay between runs to reduce predictability.
1. Accepts an alternate fee vote via JSON input file
1. Calls computefees at end of each run (attempts 3 times, spaced at 500ms)
1. Supports using delegated permissions (requires: fio.fee::setfeevote, fio.fee::setfeemultiplier, and fio.fee::computefees)

```
Usage of feevoter
  -actor string
        optional: account to use for delegated permission, alternate: $ACTOR env var
  -fees string
        optional: JSON file for overriding default fee votes, alternate: $JSON env var
  -frequency int
        optional: hours to wait between runs (does not apply to AWS Lambda) (default 2)
  -permission string
        optional: permission to use for delegated permission, alternate: $PERM env var
  -target string
        optional: target price of regaddress in USDC, alternate: $TARGET env var (default "2.0")
  -url string
        required: nodeos api url, alternate: $URL env var
  -wif string
        required: private key, alternate: $WIF env var
  -x    optional: exit after running once (does not apply to AWS Lambda,) use for running from cron
```

Here are the default fee vote values:

_note: the default fee for setfeemultiplier has been overridden to ᵮ0.1 to make frequent updates more affordable_

```
[
  {
    "end_point": "add_pub_address",
    "value": 30000000
  },
  {
    "end_point": "add_to_whitelist",
    "value": 30000000
  },
  {
    "end_point": "auth_delete",
    "value": 20000000
  },
  {
    "end_point": "auth_link",
    "value": 20000000
  },
  {
    "end_point": "auth_update",
    "value": 50000000
  },
  {
    "end_point": "burn_fio_address",
    "value": 60000000
  },
  {
    "end_point": "cancel_funds_request",
    "value": 60000000
  },
  {
    "end_point": "msig_approve",
    "value": 20000000
  },
  {
    "end_point": "msig_cancel",
    "value": 20000000
  },
  {
    "end_point": "msig_exec",
    "value": 20000000
  },
  {
    "end_point": "msig_invalidate",
    "value": 20000000
  },
  {
    "end_point": "msig_propose",
    "value": 50000000
  },
  {
    "end_point": "msig_unapprove",
    "value": 20000000
  },
  {
    "end_point": "new_funds_request",
    "value": 60000000
  },
  {
    "end_point": "proxy_vote",
    "value": 30000000
  },
  {
    "end_point": "record_obt_data",
    "value": 60000000
  },
  {
    "end_point": "register_fio_address",
    "value": 2000000000
  },
  {
    "end_point": "register_fio_domain",
    "value": 40000000000
  },
  {
    "end_point": "register_producer",
    "value": 10000000000
  },
  {
    "end_point": "register_proxy",
    "value": 1000000000
  },
  {
    "end_point": "reject_funds_request",
    "value": 30000000
  },
  {
    "end_point": "remove_all_pub_addresses",
    "value": 60000000
  },
  {
    "end_point": "remove_from_whitelist",
    "value": 30000000
  },
  {
    "end_point": "remove_pub_address",
    "value": 60000000
  },
  {
    "end_point": "renew_fio_address",
    "value": 2000000000
  },
  {
    "end_point": "renew_fio_domain",
    "value": 40000000000
  },
  {
    "end_point": "set_fio_domain_public",
    "value": 30000000
  },
  {
    "end_point": "submit_bundled_transaction",
    "value": 30000000
  },
  {
    "end_point": "submit_fee_multiplier",
    "value": 10000000
  },
  {
    "end_point": "submit_fee_ratios",
    "value": 70000000
  },
  {
    "end_point": "transfer_fio_address",
    "value": 60000000
  },
  {
    "end_point": "transfer_fio_domain",
    "value": 100000000
  },
  {
    "end_point": "transfer_tokens_pub_key",
    "value": 100000000
  },
  {
    "end_point": "unregister_producer",
    "value": 20000000
  },
  {
    "end_point": "unregister_proxy",
    "value": 20000000
  },
  {
    "end_point": "vote_producer",
    "value": 30000000
  }
]
```

