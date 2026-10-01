#!/bin/sh
# Compares the struct layouts girgen computed from GIR with what the C compiler says
# (run inside the gowt-linux container; arm64 and amd64 are both LP64 with the same field
# alignments, so the table holds for both), then runs the Go layout test.
set -e
cd "$(dirname "$0")/../.."
go run ./tooling/girgen -layout-c /tmp/layout.c -layout-expect /tmp/layout.want
gcc -w $(pkg-config --cflags gtk+-3.0 x11 cairo pangocairo) /tmp/layout.c -o /tmp/layout
/tmp/layout > /tmp/layout.got
diff /tmp/layout.want /tmp/layout.got && echo "layouts match C ($(wc -l < /tmp/layout.got) structs, $(uname -m))"
go test ./internal/gtk -run TestGeneratedLayouts
