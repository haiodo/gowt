#!/usr/bin/env bash
# Writes docs/img/<name>-<label>.png (the names docs-shots.sh writes) on the real stands: the Windows 11 arm64 UTM VM
# (.work/winvm/vm.sh) or the Ubuntu 26.04 GNOME Wayland UTM VM (.work/ubvm/vm.sh). Not Wine, not Xvfb.
# Usage: [VMTAG=catalog] [WINWD=C:\gowt\orch\gallery] [VM=path/to/vm.sh] [SCALE=2] [ONLY="coolbar example-hello app-chat"] (ONLY=apps: the seven apps) tooling/docs-shots-vm.sh windows|linux
# Run it from the repository root. The programs are cross-built for arm64, copied to the VM, run one after another in
# the logged-on desktop session and the PNGs copied back. Theme and window frame are the VM's own (Linux: light
# theme, native Wayland). A web view example is shot after its load finished (GOWT_SNAP waits for it).
# The VM display runs at SCALE (default Windows 1.5, the most its 1728x1044 display takes; GNOME 2) during the shots
# and is set back afterwards (tooling/vm-scale.sh). A catalog sample is captured at device pixels and exported at
# 2 px per point, 2x its outer size in examples/catalog/main.go; a capture below 2x is scaled up.
set -uo pipefail
label=${1:-}
export VMTAG=${VMTAG:-catalog}
tag=$VMTAG
wd=${WINWD:-C:\\gowt\\$tag} # WINWD: another work folder on the Windows VM, single backslashes
case $label in
windows) vm=${VM:-.work/winvm/vm.sh}; goos=windows; ext=.exe; script=run.cmd ;;
linux) vm=${VM:-.work/ubvm/vm.sh}; goos=linux; ext=; script=run.sh ;;
*) echo "usage: $0 windows|linux" >&2; exit 2 ;;
esac
# .work is not in a worktree checkout: fall back to the main checkout next to .claude/worktrees.
[ -x "$vm" ] || vm=$(git rev-parse --path-format=absolute --git-common-dir)/../$vm
out=$(pwd)/docs/img
bin=$(pwd)/bin/docs/vm-$label
rm -rf "$bin"; mkdir -p "$bin"
scale=${SCALE:-$([ "$label" = windows ] && echo 1.5 || echo 2)}
prevscale=$(tooling/vm-scale.sh "$label" "$scale") || exit 1
trap 'tooling/vm-scale.sh "$label" "$prevscale" >/dev/null' EXIT

jobs=() # "<png stem> <program> [catalog widget]"
for e in $(ls examples | grep -vx -e controlexample -e catalog -e apps); do
	GOOS=$goos GOARCH=arm64 go build -o "$bin/$e$ext" "./examples/$e" || exit 1
	jobs+=("example-$e $e")
done
GOOS=$goos GOARCH=arm64 go build -o "$bin/catalog$ext" ./examples/catalog || exit 1
for w in $(sed -n 's/^	{"\([a-z]*\)", .*/\1/p' examples/catalog/main.go); do jobs+=("$w catalog $w"); done
# examples/apps: app-<name>, light appearance, opened at kit.SnapWidth x kit.SnapHeight (examples/apps/internal/kit).
for a in $(ls examples/apps | grep -vx internal); do
	GOOS=$goos GOARCH=arm64 go build -o "$bin/app-$a$ext" "./examples/apps/$a" || exit 1
	jobs+=("app-$a app-$a")
done
[ "${ONLY:-}" = apps ] && ONLY=$(printf 'app-%s ' $(ls examples/apps | grep -vx internal))

{
	if [ "$label" = windows ]; then
		echo '@echo off'; echo "cd /d $wd"; echo 'if not exist out mkdir out';
	else
		echo 'mkdir -p out; rm -f out/*'
	fi
	for j in "${jobs[@]}"; do
		set -- $j
		if [ "$label" = windows ]; then
			[ -n "${ONLY:-}" ] && [[ " $ONLY " != *" $1 "* ]] && continue
			[ -n "${RAWSIZE:-}" ] && echo "set GOWT_SNAP_SIZE=0x0"
			case $1 in app-*) echo "set GOWT_APPEARANCE=light";; *) echo "set GOWT_APPEARANCE=";; esac
			echo "set GOWT_SNAP=$wd\\out\\$1-$label.png"
			echo "$2.exe ${3:-} 2>out\\$1.log"
		else
			[ -n "${ONLY:-}" ] && [[ " $ONLY " != *" $1 "* ]] && continue
			# Native Wayland reads a WebKitGTK window back as empty; the web view examples are shot through Xwayland.
			x=${RAWSIZE:+GOWT_SNAP_SIZE=0x0}; case $2 in browser|webview) x="$x GDK_BACKEND=x11 GDK_SCALE=$scale";; app-*) x="$x GOWT_APPEARANCE=light";; esac
			# GTK_THEME pins the light theme whatever the desktop's setting is; an empty capture (window not painted yet) is an error and is shot again later.
			# The apps lay out slower than a catalog sample: at 1000 ms three of them were still at their minimum size.
			delays="1000 3000 6000"; case $1 in app-*) delays="3000 6000";; esac
			echo "for d in $delays; do $x GTK_THEME=Adwaita:light GOWT_SNAP_DELAY=\$d GOWT_SNAP=\$PWD/out/$1-$label.png timeout 90 ./$2 ${3:-} >out/$1.log 2>&1; [ -s out/$1-$label.png ] && break; done"
		fi
	done
} >"$bin/$script"

if [ "$label" = windows ]; then
	"$vm" ssh "Remove-Item -Recurse -Force $wd\\out -ErrorAction SilentlyContinue; New-Item -ItemType Directory -Force $wd | Out-Null"
	"$vm" put "$bin"/*.exe "$bin/$script" "$wd"
	"$vm" gui "cd /d $wd && call $script" 3000 || exit 1
	"$vm" get "${wd//\\//}/out" "$bin/got"
else
	"$vm" ssh 'rm -rf out'
	"$vm" put "$bin"/* "~/gowt/$tag/"
	"$vm" gui 'chmod +x * 2>/dev/null; sh run.sh'
	"$vm" get "~/gowt/$tag/out" "$bin/got"
fi
# Which stand made the pictures: docs/img/source-<label>.txt, rewritten by a full run (a run with ONLY leaves it alone).
if [ -z "${ONLY:-}" ]; then
	if [ "$label" = windows ]; then
		stand=$("$vm" ssh '(Get-CimInstance Win32_OperatingSystem | % { $_.Caption + " build " + $_.BuildNumber + " " + $_.OSArchitecture })' | tr -d '\r')
		stand="Windows in a UTM virtual machine: $stand; WebView2 for the web examples; whole window with its DWM frame copied from the screen"
	else
		stand=$("$vm" ssh '. /etc/os-release; echo "$PRETTY_NAME $(uname -m)"')
		stand="Ubuntu GNOME in a UTM virtual machine with a GPU, native Wayland, light theme (GTK_THEME=Adwaita:light): $stand; web view examples through Xwayland; client-side title bar, shadow trimmed"
	fi
	# The scale the programs saw, from the line GOWT_SNAP prints for a catalog sample.
	seen=$(sed -n 's/.*capture scale \([0-9.]*\),.*/\1/p' "$bin"/got/*.log | sort | uniq -c | sort -rn | head -1 | awk '{ print $2 }')
	printf 'stand: %s\ndisplay scale: %s (requested %s; catalog samples captured at device pixels, exported at 2 px per point)\nsource: tooling/docs-shots-vm.sh %s\ncommit: %s\ndate: %s\n' "$stand" "${seen:-unknown}" "$scale" "$label" "$(git rev-parse --short HEAD)" "$(date +%F)" >"$out/source-$label.txt"
fi
fail=0
for j in "${jobs[@]}"; do
	set -- $j
	[ -n "${ONLY:-}" ] && [[ " $ONLY " != *" $1 "* ]] && continue
	if [ -s "$bin/got/$1-$label.png" ]; then cp "$bin/got/$1-$label.png" "$out/"; echo "$1-$label.png"; else echo "FAILED $1-$label.png"; fail=1; fi
done
exit $fail
