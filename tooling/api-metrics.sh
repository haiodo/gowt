#!/usr/bin/env bash
# Counts, per program built on the gowt facade, how much code a screen takes and how often it leaves
# the facade: non-blank lines, lines with Unwrap(), lines with swt., local placement helpers
# (`fillX := gowt.Cell(...)`) and GridCell literals. Arguments are directories; without them it
# measures examples/* (controlexample is translated SWT, not facade code), cmd/* that import the
# facade, and $DOCKERDESK (default ../dockerdesk) when it exists.
set -euo pipefail
cd "$(dirname "$0")/.."
dirs=("$@")
if [ ${#dirs[@]} -eq 0 ]; then
	for d in examples/*/; do
		[ "$d" = examples/controlexample/ ] || dirs+=("${d%/}")
	done
	for d in $(grep -l '"github.com/haiodo/gowt"' cmd/*/*.go | xargs -n1 dirname | sort -u); do
		dirs+=("$d")
	done
	dd=${DOCKERDESK:-../dockerdesk}
	[ -d "$dd" ] && dirs+=("$dd")
fi
count() { cat "${files[@]}" | grep -cE "$1" || true; }
printf '%-26s %5s %6s %6s %5s %7s %8s\n' program files lines Unwrap swt. helpers GridCell
tot=(0 0 0 0 0 0)
for d in "${dirs[@]}"; do
	files=()
	while IFS= read -r f; do files+=("$f"); done < <(find "$d" -name '*.go' ! -name '*_test.go' | sort)
	[ ${#files[@]} -gt 0 ] || continue
	row=("${#files[@]}" "$(count '[^[:space:]]')" "$(count 'Unwrap\(\)')" "$(count '\bswt\.')" \
		"$(count '^[[:space:]]*(var[[:space:]]+)?[[:alnum:]_]+[[:space:]]*:?=[[:space:]]*gowt\.Cell\(')" \
		"$(count 'GridCell\{')")
	printf '%-26s %5d %6d %6d %5d %7d %8d\n' "$(basename "$d")" "${row[@]}"
	for i in 0 1 2 3 4 5; do tot[i]=$((tot[i] + row[i])); done
done
printf '%-26s %5d %6d %6d %5d %7d %8d\n' total "${tot[@]}"
