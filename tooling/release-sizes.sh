#!/usr/bin/env bash
# Prints the size of each bin/release binary and warns when it grew more than 5% over tooling/sizes.txt.
# With -update it rewrites tooling/sizes.txt from the current binaries instead.
set -euo pipefail
cd "$(dirname "$0")/.."
ref=tooling/sizes.txt
new=$(mktemp)
for f in bin/release/*; do
	printf '%s %d\n' "$(basename "$f")" "$(stat -f %z "$f")"
done >"$new"
if [ "${1:-}" = "-update" ]; then mv "$new" "$ref"; exit 0; fi
while read -r name size; do
	old=$(awk -v n="$name" '$1 == n {print $2}' "$ref" 2>/dev/null || true)
	printf '%10d  %s' "$size" "$name"
	if [ -n "$old" ] && [ $((size * 100)) -gt $((old * 105)) ]; then
		printf '  WARNING: +%d%% over recorded %d' $(((size - old) * 100 / old)) "$old"
	fi
	printf '\n'
done <"$new"
rm -f "$new"
