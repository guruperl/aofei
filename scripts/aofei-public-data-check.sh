#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT"

failed=0

check_pattern() {
	local label="$1"
	local pattern="$2"
	if git grep -nIE "$pattern" -- .; then
		printf 'public-data-check: tracked files contain %s\n' "$label" >&2
		failed=1
	fi
}

check_pattern "AWS access-key identifiers" 'A(KIA|SIA)[0-9A-Z]{16}'
check_pattern "private key material" 'BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY'
check_pattern "a private home path" '/home/'peter
check_pattern "customer email domains" 'kinet'"\\.com"
check_pattern "retired customer exchange domains" 'leadsadx'"-trade\\.com|example-dsp|example-ad"
check_pattern "customer account identifiers" 'peter_'"00[0-9]"

if git ls-files '*.docx' | grep -q .; then
	echo "public-data-check: DOCX customer sources must not be tracked" >&2
	failed=1
fi

if git ls-files '.local/*' 'etc/*.local.json' | grep -q .; then
	echo "public-data-check: generated runtime configuration is tracked" >&2
	failed=1
fi

if git ls-files 'backup/*' | grep -vFx 'backup/README.md' | grep -q .; then
	echo "public-data-check: backup/ may contain only its policy README" >&2
	failed=1
fi

mutable_tables='add_address|admin|adv|adv_balance|adv_campaign|adv_creative|adv_ip|adv_item|adv_media|cron_halfhour|ledger_adv|ledger_log|ledger_pub_adv|ledger_pub|pub|pub_site|pub_slot'
if rg -n "^INSERT INTO \`(${mutable_tables})\`" etc/step4_init.sql; then
	echo "public-data-check: the schema baseline contains mutable business rows" >&2
	failed=1
fi

for sample in etc/samples/sample_100.json etc/samples/sample_adm.json \
	etc/samples/sample_bid.json etc/samples/sample_native.json \
	etc/samples/sample_resp.json etc/samples/sample_win.json
do
	jq -e . "$sample" >/dev/null
done

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "Public-data guard passed."
