#!/usr/bin/env bash
# Sets the display scale of the Windows 11 or Ubuntu GNOME UTM VM (.work/winvm/vm.sh, .work/ubvm/vm.sh) and prints
# the previous one, so a caller can restore it: tooling/vm-scale.sh windows|linux <factor> (1, 1.25, 1.5, 2, ...).
# Windows: tooling/win-dpi.ps1 in the logged-on session, applied at once. The 1728x1044 display follows the size of
# the UTM window, so the resolution cannot be raised for 200%: Windows takes 150% at most there (the shots record
# the scale the program saw). Linux: Mutter's ApplyMonitorsConfig (temporary, gone with the session).
# Needs VMTAG for the work folder, as docs-shots-vm.sh does.
set -euo pipefail
os=${1:-}; factor=${2:-}
[ -n "$factor" ] || { echo "usage: $0 windows|linux <factor>" >&2; exit 2; }
export VMTAG=${VMTAG:-catalog}
tag=$VMTAG
wd=${WINWD:-C:\\gowt\\$tag} # WINWD: another work folder on the Windows VM, single backslashes
case $os in
windows) vm=${VM:-.work/winvm/vm.sh} ;;
linux) vm=${VM:-.work/ubvm/vm.sh} ;;
*) echo "usage: $0 windows|linux <factor>" >&2; exit 2 ;;
esac
[ -x "$vm" ] || vm=$(git rev-parse --path-format=absolute --git-common-dir)/../$vm

if [ "$os" = windows ]; then
	percent=$(awk -v f="$factor" 'BEGIN { printf "%d", f * 100 }')
	"$vm" ssh "New-Item -ItemType Directory -Force $wd | Out-Null"
	"$vm" put tooling/win-dpi.ps1 "$wd"
	run() { # <get|set percent>
		"$vm" gui "powershell -NoProfile -ExecutionPolicy Bypass -File $wd\\win-dpi.ps1 $* > $wd\\dpi.txt 2>&1" 60 >/dev/null
		"$vm" ssh "Get-Content $wd\\dpi.txt" | tr -d '\r'
	}
	prev=$(run get | sed -n 's/^current=\([0-9]*\).*/\1/p')
	[ -n "$prev" ] || { echo "cannot read the Windows display scale" >&2; exit 1; }
	[ "$prev" = "$percent" ] || run set "$percent" >/dev/null
	awk -v p="$prev" 'BEGIN { print p / 100 }'
else
	st=$("$vm" gui 'gdbus call --session --dest org.gnome.Mutter.DisplayConfig --object-path /org/gnome/Mutter/DisplayConfig --method org.gnome.Mutter.DisplayConfig.GetCurrentState')
	serial=$(sed 's/^(uint32 \([0-9]*\),.*/\1/' <<<"$st")
	conn=$(sed "s/^(uint32 [0-9]*, \[(('\([^']*\)'.*/\1/" <<<"$st")
	mode=$(sed -n "s/.*('\([0-9x@.]*\)', [0-9]*, [0-9]*, [0-9.]*, [0-9.]*, \[[^]]*\], {'is-current'.*/\1/p" <<<"$st")
	prev=$(sed -n 's/.*\[(0, 0, \([0-9.]*\), uint32.*/\1/p' <<<"$st")
	[ -n "$prev" ] && [ -n "$mode" ] || { echo "cannot read the GNOME monitor state" >&2; exit 1; }
	"$vm" gui "gdbus call --session --dest org.gnome.Mutter.DisplayConfig --object-path /org/gnome/Mutter/DisplayConfig --method org.gnome.Mutter.DisplayConfig.ApplyMonitorsConfig $serial 1 \"[(0, 0, $factor, uint32 0, true, [('$conn', '$mode', {})])]\" \"{}\"" >/dev/null
	echo "$prev"
fi
