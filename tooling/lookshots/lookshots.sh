#!/usr/bin/env bash
# Builds the look-check apps (examples/apps), captures them on the stands with GOWT_SNAP and writes one
# side-by-side sheet per app. See docs/look-check.md.
# Usage: tooling/lookshots/lookshots.sh <step>... [-- app...]
# Steps: build  mac  win  linux-vm  linux-docker  sheet  remote (= build win linux-vm linux-docker sheet)
# mac opens real windows on this Mac: the owner runs it, never an agent.
# Env: OUT (default <repo>/.work/look-apps), WINTAG, VMHOST for the Ubuntu VM helper, LINUX_CTR and LINUX_PORT
# for the Docker stand, WINVM and UBVM for the helper scripts.
set -uo pipefail
cd "$(dirname "$0")/../.."
root=$(cd "$(git rev-parse --git-common-dir)/.." && pwd)
OUT=${OUT:-$root/.work/look-apps}
WINVM=${WINVM:-$root/.work/winvm/vm.sh}
UBVM=${UBVM:-$root/.work/ubvm/vm.sh}
LINUX_CTR=${LINUX_CTR:-gowt-lookapps}
LINUX_PORT=${LINUX_PORT:-6091}
export VMTAG=${VMTAG:-lookapps}

steps=(); apps=()
while [ $# -gt 0 ]; do
	[ "$1" = -- ] && { shift; apps=("$@"); break; }
	steps+=("$1"); shift
done
[ ${#apps[@]} -eq 0 ] && apps=($(ls examples/apps | grep -vx internal))
[ ${#steps[@]} -eq 0 ] && { sed -n 2,9p "$0"; exit 2; }

build() { # <goos> <goarch> <dir> [ext]
	local d=$OUT/bin/$3
	mkdir -p "$d"
	for a in "${apps[@]}"; do
		GOOS=$1 GOARCH=$2 go build -o "$d/$a${4:-}" "./examples/apps/$a" || return 1
	done
}

# One capture per app: fixed size comes from the app, GOWT_SNAP_DELAY lets the first layout settle.
mac() {
	mkdir -p "$OUT/shots/macos"
	build darwin arm64 darwin-arm64 || return 1
	for a in "${apps[@]}"; do
		GOWT_SNAP=$OUT/shots/macos/$a.png GOWT_SNAP_DELAY=${DELAY:-1500} GOWT_SNAP_SCREEN=${MAC_SCREEN:-1} \
			perl -e 'alarm 60; exec @ARGV' "$OUT/bin/darwin-arm64/$a" >"$OUT/shots/macos/$a.log" 2>&1
		[ -s "$OUT/shots/macos/$a.png" ] && echo "macos $a" || echo "FAILED macos $a"
	done
}

win() {
	local d='C:\gowt\'$VMTAG s=$OUT/shots/windows
	mkdir -p "$s"
	rm -rf "$s"/*.png
	local cmd=$OUT/run-win.cmd
	{ echo '@echo off'; echo "set GOWT_SNAP_DELAY=${DELAY:-1500}"; echo "mkdir $d\\shots 2>nul"
	  for a in "${apps[@]}"; do echo "set GOWT_SNAP=$d\\shots\\$a.png"; echo "$d\\bin\\$a.exe"; done; } >"$cmd"
	"$WINVM" ssh "New-Item -ItemType Directory -Force $d\\bin | Out-Null; Remove-Item $d\\shots -Recurse -ErrorAction SilentlyContinue" >/dev/null
	"$WINVM" put "$OUT/bin/windows-arm64/"*.exe "$cmd" "$d\\bin\\" && "$WINVM" ssh "Move-Item -Force $d\\bin\\run-win.cmd $d\\run.cmd" >/dev/null || return 1
	"$WINVM" gui "call $d\\run.cmd" 600 || return 1
	for a in "${apps[@]}"; do "$WINVM" get "C:/gowt/$VMTAG/shots/$a.png" "$s/$a.png" && echo "windows $a" || echo "FAILED windows $a"; done
}

linux_vm() {
	local s=$OUT/shots/linux-vm
	mkdir -p "$s"
	"$UBVM" ssh 'rm -rf bin shots; mkdir -p shots' >/dev/null
	"$UBVM" put "$OUT"/bin/linux-arm64/* '~/gowt/'$VMTAG'/bin' || return 1
	"$UBVM" gui "cd ~/gowt/$VMTAG; for a in ${apps[*]}; do GOWT_SNAP=shots/\$a.png GOWT_SNAP_DELAY=${DELAY:-1500} timeout 60 ./bin/\$a >shots/\$a.log 2>&1; done"
	for a in "${apps[@]}"; do "$UBVM" get "gowt/$VMTAG/shots/$a.png" "$s/$a.png" && echo "linux-vm $a" || echo "FAILED linux-vm $a"; done
}

linux_docker() {
	local s=$OUT/shots/linux-docker
	mkdir -p "$s"
	make -s linux-vnc LINUX_CTR="$LINUX_CTR" LINUX_PORT="$LINUX_PORT" >/dev/null || return 1
	docker exec "$LINUX_CTR" sh -c 'rm -rf /tmp/lookapps; mkdir -p /tmp/lookapps/shots' || return 1
	docker cp "$OUT/bin/linux-arm64/." "$LINUX_CTR:/tmp/lookapps/bin" || return 1
	docker exec "$LINUX_CTR" sh -c "cd /tmp/lookapps; for a in ${apps[*]}; do GOWT_SNAP=shots/\$a.png GOWT_SNAP_DELAY=${DELAY:-1500} timeout 60 ./bin/\$a >shots/\$a.log 2>&1; done"
	for a in "${apps[@]}"; do docker cp "$LINUX_CTR:/tmp/lookapps/shots/$a.png" "$s/$a.png" 2>/dev/null && echo "linux-docker $a" || echo "FAILED linux-docker $a"; done
}

for st in "${steps[@]}"; do
	case $st in
	build) build darwin arm64 darwin-arm64 && build windows arm64 windows-arm64 .exe && build linux arm64 linux-arm64 ;;
	mac) mac ;;
	win) build windows arm64 windows-arm64 .exe && win ;;
	linux-vm) build linux arm64 linux-arm64 && linux_vm ;;
	linux-docker) build linux arm64 linux-arm64 && linux_docker ;;
	sheet) python3 tooling/lookshots/sheet.py "$OUT" "${apps[@]}" ;;
	remote) "$0" win linux-vm linux-docker sheet -- "${apps[@]}" ;;
	*) echo "unknown step $st" >&2; exit 2 ;;
	esac || echo "step $st failed" >&2
done
