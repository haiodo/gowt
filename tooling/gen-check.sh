#!/usr/bin/env bash
# Regenerates every platform (make gen for cocoa, gtk and win32) and fails if the committed generated files
# differ: stale output, or a generator change committed without its regenerated output.
# SWT_REPO and UI_REPO must be at the commits in tooling/source-pins.env.
set -euo pipefail
cd "$(dirname "$0")/.."
. tooling/source-pins.env

SWT_REPO="${SWT_REPO:-$HOME/Develop/repos/eclipse.platform.swt}"
UI_REPO="${UI_REPO:-$HOME/Develop/repos/eclipse.platform.ui}"
export SWT_REPO UI_REPO
[ "$(git -C "$SWT_REPO" rev-parse HEAD)" = "$SWT_SHA" ] || { echo "gen-check: $SWT_REPO is not at $SWT_SHA" >&2; exit 2; }
[ "$(git -C "$UI_REPO" rev-parse HEAD)" = "$UI_SHA" ] || { echo "gen-check: $UI_REPO is not at $UI_SHA" >&2; exit 2; }

paths=(swt jface browser internal tests examples)
if [ -n "$(git status --porcelain -- "${paths[@]}")" ]; then
	echo "gen-check: commit or stash changes under ${paths[*]} first, the check compares against the working tree" >&2
	exit 2
fi

for p in cocoa gtk win32; do
	echo "gen-check: PLATFORM=$p"
	PLATFORM=$p bash tooling/port.sh > /dev/null
done

if [ -n "$(git status --porcelain -- "${paths[@]}")" ]; then
	echo "gen-check: committed generated code differs from what the generator produces:" >&2
	git status --short -- "${paths[@]}" >&2
	exit 1
fi
echo "gen-check: generated code is up to date"
