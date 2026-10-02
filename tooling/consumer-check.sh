#!/usr/bin/env bash
# Builds a module outside this repository (example.com/app, replace -> this checkout) for darwin, windows and linux:
# what a user gets from `go get`. CONSUMER=<dir> picks where it lives (default: ../consumer next to the checkout).
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
dir="${CONSUMER:-$repo/../consumer}"
mkdir -p "$dir" && cd "$dir"
cat > main.go <<'GO'
package main

import (
	"runtime"

	"github.com/haiodo/gowt/swt"
	_ "github.com/haiodo/gowt/winmanifest"
)

func init() { runtime.LockOSThread() }

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetLayout(swt.NewFillLayout())
	swt.NewButton(shell, swt.PUSH).SetText("Hello")
	shell.Pack()
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
GO
[ -f go.mod ] || go mod init example.com/app
go mod edit -go="$(sed -n 's/^go //p' "$repo/go.mod")" -require=github.com/haiodo/gowt@v0.0.0 -replace=github.com/haiodo/gowt="$repo"
go mod tidy
for t in darwin/arm64 windows/amd64 windows/arm64 linux/amd64; do
	CGO_ENABLED=0 GOOS=${t%/*} GOARCH=${t#*/} go build -o /dev/null . && echo "ok $t"
done
