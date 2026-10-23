#!/usr/bin/env bash
# The QA server (package qa, -tags gowtqa) must be in a QA build and in no other: builds the fixture app
# both ways and looks for the server's symbols. qatest must link no toolkit package. The tagged code must vet on every OS.
set -euo pipefail
cd "$(dirname "$0")/.."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

app=./qa/qatest/testdata/app
go build -o "$tmp/plain" $app
go build -tags gowtqa -o "$tmp/qa" $app
go tool nm "$tmp/plain" | sort > "$tmp/plain.sym"
go tool nm "$tmp/qa" | sort > "$tmp/qa.sym"

if grep -E 'gowt/qa\.\(\*server\)|gowt/qa\.listen|gowt/internal/qatree\.' "$tmp/plain.sym"; then
	echo "qa-check: the build without -tags gowtqa contains the QA server" >&2
	exit 1
fi
grep -q 'gowt/qa\.(\*server)\.run' "$tmp/qa.sym" || { echo "qa-check: the build with -tags gowtqa lacks the QA server" >&2; exit 1; }

if go list -deps ./qa/qatest | grep -E 'gowt/(swt|internal/(cocoa|gtk|win32))'; then
	echo "qa-check: qa/qatest links a toolkit package" >&2
	exit 1
fi

for os in darwin windows linux; do
	GOOS=$os go vet -tags gowtqa . ./qa/... ./internal/qatree
done
echo "qa-check: ok"
