#!/usr/bin/env bash
# Writes docs/img/<widget>-<label>.png for every sample of examples/catalog and
# docs/img/example-<name>-<label>.png for every other example, using GOWT_SNAP (see docs/screenshots.md).
# Usage: [GOOS=windows] [RUN="wine --bottle gowt"] [EXT=.exe] [PATHPFX=Z:] tooling/docs-shots.sh <label>
# Run it from the repository root. RUN prefixes the executable, PATHPFX the output path as the program sees it.
set -uo pipefail
label=$1
out=${OUT:-docs/img}
bin=bin/docs/$label
ext=${EXT:-}
mkdir -p "$out" "$bin"
out=$(cd "$out" && pwd)
fail=0

shoot() { # <png name> <executable> [args...]
	local png=$1 exe=$2; shift 2
	rm -f "$out/$png"
	# The alarm is the backstop for a stuck window; GOWT_SNAP ends a healthy run by itself.
	GOWT_SNAP="${PATHPFX:-}$out/$png" perl -e 'alarm 60; exec @ARGV' ${RUN:-} "$exe" "$@" >"$bin/last.log" 2>&1
	if [ ! -s "$out/$png" ]; then
		echo "FAILED $png"; sed 's/^/  /' "$bin/last.log" | tail -5; fail=1
	else
		echo "$png"
	fi
}

for e in $(ls examples | grep -vx -e controlexample -e catalog); do
	go build -o "$bin/$e$ext" "./examples/$e" || { fail=1; continue; }
	delay=800; case $e in webview|browser) delay=4000;; esac
	GOWT_SNAP_DELAY=$delay shoot "example-$e-$label.png" "$(pwd)/$bin/$e$ext"
done

go build -o "$bin/catalog$ext" ./examples/catalog || exit 1
for w in $(sed -n 's/^	{"\([a-z]*\)", func.*/\1/p' examples/catalog/main.go); do
	shoot "$w-$label.png" "$(pwd)/$bin/catalog$ext" "$w"
done
exit $fail
