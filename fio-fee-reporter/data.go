package ffr

import "sync"

var (
	feeMap = map[string]string {
		"add_bundled_transactions": "addbundles",
		"add_pub_address": "addaddress",
		"cancel_funds_request": "cancelfndreq",
		"remove_pub_address": "remaddress",
		"msig_unapprove": "unapprove",
		"vote_producer": "voteproducer",
		"register_fio_domain": "regdomain",
		"register_producer": "regproducer",
		"msig_approve": "approve",
		"submit_bundled_transaction": "bundlevote",
		"msig_cancel": "cancel",
		"auth_link": "linkauth",
		"new_funds_request": "newfundsreq",
		"record_obt_data": "recordobt",
		"auth_update": "updateauth",
		"set_fio_domain_public": "setdomainpub",
		"submit_fee_multiplier": "setfeemult",
		"unregister_proxy": "unregproxy",
		"submit_fee_ratios": "setfeevote",
		"burn_fio_address": "burnaddress",
		"msig_invalidate": "invalidate",
		"register_proxy": "regproxy",
		"remove_all_pub_addresses": "remalladdr",
		"renew_fio_address": "renewaddress",
		"renew_fio_domain": "renewdomain",
		"msig_exec": "exec",
		"msig_propose": "propose",
		"unregister_producer": "unregprod",
		"auth_delete": "deleteauth",
		"register_fio_address": "regaddress",
		"transfer_tokens_pub_key": "trnsfiopubky",
		"transfer_locked_tokens": "trnsloctoks",
		"transfer_fio_domain": "xferdomain",
		"reject_funds_request": "rejectfndreq",
		"proxy_vote": "voteproxy",
		"transfer_fio_address": "xferaddress",
	}
	feeMapMux = sync.RWMutex{}
)
