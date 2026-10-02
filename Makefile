export CGO_ENABLED := 0

SWT_REPO ?= $(HOME)/Develop/repos/eclipse.platform.swt
export SWT_REPO

# Platform port.sh translates: cocoa now, win32 and gtk are prepared slots.
PLATFORM ?= $(if $(filter linux,$(shell go env GOOS)),gtk,cocoa)
export PLATFORM

# Per-OS gate: tests/expected.txt is darwin's, tests/expected_<goos>.txt another OS's.
HOSTOS := $(shell go env GOOS)
EXPECTED ?= tests/expected$(if $(filter-out darwin,$(HOSTOS)),_$(HOSTOS)).txt

CMDS := $(notdir $(wildcard cmd/*))
BIN  := bin

.PHONY: app jfacetest test-jface test-jface-update gtk-gen winmanifest win-build win-probe win-hello win-swttest win-swttest-update win-snap-check win-snap-update all gen build release release-sizes-update vet test check xcheck api-check clean test-swt test-swt-update snap-check snap-update run-% $(CMDS)

all: check build

# Rebuilds the translator and regenerates swt/, internal/<PLATFORM>/, examples/ and tests/swttests/ from the SWT sources.
gen:
	bash tooling/port.sh

build: $(CMDS)

$(CMDS):
	go build -o $(BIN)/$@ ./cmd/$@

# Stripped builds into bin/release/ with a size per binary (warns over +5% vs tooling/sizes.txt) (pclntab stays: stack traces).
release:
	@mkdir -p $(BIN)/release
	@for c in $(CMDS); do \
		go build -trimpath -ldflags='-s -w' -o $(BIN)/release/$$c ./cmd/$$c || exit 1; \
	done
	@bash tooling/release-sizes.sh

# macOS app bundles of the release binaries, e.g. make app APPS="minibrowser webviewdemo".
APPS ?= minibrowser webviewdemo jfacedemo controlexample
app: release
	@for c in $(APPS); do bash tooling/darwin/mkapp.sh $(BIN)/release/$$c $(BIN)/release || exit 1; done

# Records the current bin/release sizes as the reference for the 5% growth warning.
release-sizes-update: release
	@bash tooling/release-sizes.sh -update

vet:
	go vet ./...

test:
	go test ./...

check: vet test xcheck api-check

# Platform-neutral code must build for every OS and must not import a platform's PI package; the
# platform-specific rest (swt, cmd, ...) builds only where that platform's port exists. jface is one source for every OS: it builds on all three.
XCHECK_PKGS := ./internal/jrt ./internal/junit ./internal/snapcmp ./cmd/snapcheck ./tooling/apidump ./examples/controlexample/res
xcheck:
	@for os in windows linux; do GOOS=$$os go build $(XCHECK_PKGS) || exit 1; done
	@for os in darwin windows linux; do GOOS=$$os go build ./jface || exit 1; done
	@bad=$$(grep -lE '"github.com/haiodo/gowt/internal/(cocoa|win32|gtk)"' $$(ls swt/*.go jface/*.go examples/*/*.go tests/swttests/*.go cmd/*/*.go | grep -vE '_(darwin|windows|linux)(_test)?\.go$$') || true); \
	if [ -n "$$bad" ]; then echo "platform import in files without a GOOS suffix:"; echo "$$bad"; exit 1; fi

# Exported swt API of the platforms in tooling/apidump/platforms.txt must agree (platform-only.txt lists the exceptions).
api-check:
	go run ./tooling/apidump -check

# Translated SWT JUnit tests on the main thread (cmd/swttest), gated by tests/expected.txt: fails on a
# regression or an undescribed failure. SWTTEST_FLAGS e.g. -run GC (the gate then checks only those).
test-swt: swttest
	./$(BIN)/swttest -expected $(EXPECTED) $(SWTTEST_FLAGS)

# JFace's translated JUnit tests (tests/jfacetests): the same runner built with -tags jface, gated by tests/expected_jface[_os].txt.
JEXPECTED ?= tests/expected_jface$(if $(filter-out darwin,$(HOSTOS)),_$(HOSTOS)).txt
jfacetest:
	go build -tags jface -o $(BIN)/jfacetest ./cmd/swttest

test-jface: jfacetest
	./$(BIN)/jfacetest -expected $(JEXPECTED) $(SWTTEST_FLAGS)

test-jface-update: jfacetest
	./$(BIN)/jfacetest -update $(JEXPECTED) $(SWTTEST_FLAGS)

# Rewrites the expected file from a full run; a new failure comes out as UNDESCRIBED until its cause is written.
test-swt-update: swttest
	./$(BIN)/swttest -update $(EXPECTED) $(SWTTEST_FLAGS)

# Per-OS references: tests/snapshots is darwin's, tests/snapshots_<goos> another OS's.
SNAPDIR ?= tests/snapshots$(if $(filter-out darwin,$(HOSTOS)),_$(HOSTOS))

# ControlExample tab snapshots against $(SNAPDIR) (cmd/snapcheck: per-pixel threshold + allowed
# fraction). Skips with a message when the environment (meta.txt: scale, OS and GTK version, ...) differs from the references.
# Failures leave diff images in bin/snap/diff.
snap-check: controlexample snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	./$(BIN)/controlexample -snap $(BIN)/snap/got > $(BIN)/snap/run.log
	./$(BIN)/snapcheck -ref $(SNAPDIR) -got $(BIN)/snap/got -diff $(BIN)/snap/diff

# Rewrites $(SNAPDIR) (PNGs + meta.txt) from a fresh run.
snap-update: controlexample snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	./$(BIN)/controlexample -snap $(BIN)/snap/got > $(BIN)/snap/run.log
	./$(BIN)/snapcheck -update -ref $(SNAPDIR) -got $(BIN)/snap/got

run-%: %
	./$(BIN)/$*

clean:
	rm -rf $(BIN) tooling/j2go/target

# Windows port (CrossOver bottle "gowt"): cross-build the exes, run them under Wine. win-probe opens no window;
# the others do. Paths are absolute (Wine reads the host file system as drive Z:).
WINE ?= /Applications/CrossOver.app/Contents/SharedSupport/CrossOver/bin/wine
WINBOTTLE ?= gowt
WINBIN := $(BIN)/windows
WINCMDS := hello swttest controlexample winprobe jfacedemo

# winmanifest/*.syso is app.manifest (comctl32 v6, per-monitor DPI v2) as a COFF resource; cmd/* get it by importing
# winmanifest. tooling/mksyso writes the objects itself, no windres needed. Rebuild after editing the manifest.
winmanifest:
	@for a in amd64 arm64; do go run ./tooling/mksyso -arch $$a -manifest winmanifest/app.manifest -o winmanifest/manifest_windows_$$a.syso || exit 1; done

win-build:
	@mkdir -p $(WINBIN)
	@for c in $(WINCMDS); do GOOS=windows GOARCH=amd64 go build -o $(WINBIN)/$$c.exe ./cmd/$$c || exit 1; done

win-probe: win-build
	$(WINE) --bottle $(WINBOTTLE) $(abspath $(WINBIN))/winprobe.exe

win-hello: win-build
	$(WINE) --bottle $(WINBOTTLE) $(abspath $(WINBIN))/hello.exe

# The translated SWT JUnit tests against tests/expected_windows.txt (win-swttest-update writes it from a full run).
win-swttest: win-build
	$(WINE) --bottle $(WINBOTTLE) $(abspath $(WINBIN))/swttest.exe -expected tests/expected_windows.txt $(SWTTEST_FLAGS)

win-swttest-update: win-build
	$(WINE) --bottle $(WINBOTTLE) $(abspath $(WINBIN))/swttest.exe -update tests/expected_windows.txt $(SWTTEST_FLAGS)

# Snapshots of the Windows port: the exe runs in the bottle (Z: is the host file system), snapcheck on the host.
# References are tests/snapshots_windows; meta.txt (Wine version, DPI, theme, screen) makes snap-check SKIP elsewhere.
WINSNAPDIR := tests/snapshots_windows
# The exe stops itself 20 s into a stuck step; the alarm is the backstop (no coreutils timeout on macOS).
WINSNAPRUN = GOWT_TRACE_DISPATCH=1 perl -e 'alarm 120; exec @ARGV' $(WINE) --bottle $(WINBOTTLE) $(abspath $(WINBIN))/controlexample.exe -snap Z:$(abspath $(BIN)/snap/got) > $(BIN)/snap/run.log || { echo "controlexample.exe failed or stalled (see above and $(BIN)/snap/run.log)"; pkill -f controlexample.exe; exit 1; }

win-snap-check: win-build snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	$(WINSNAPRUN)
	./$(BIN)/snapcheck -ref $(WINSNAPDIR) -got $(BIN)/snap/got -diff $(BIN)/snap/diff

win-snap-update: win-build snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	$(WINSNAPRUN)
	./$(BIN)/snapcheck -update -ref $(WINSNAPDIR) -got $(BIN)/snap/got

# Linux GUI stand (Docker + Xvfb + noVNC), see README "Linux stand".
LINUX_IMG = gowt-linux
LINUX_CTR = gowt-linux

linux-image:
	docker build -t $(LINUX_IMG) tooling/linux

# Idempotent: starts the stand container and prints the noVNC URL.
linux-vnc:
	@docker ps -q -f name=^$(LINUX_CTR)$$ | grep -q . || { docker rm -f $(LINUX_CTR) >/dev/null 2>&1; \
	  docker run -d --name $(LINUX_CTR) -p 6080:6080 -v $(CURDIR):/src -v gowt-gocache:/gocache -v gowt-gomod:/gomod $(LINUX_IMG) >/dev/null; sleep 2; }
	@echo http://localhost:6080/vnc.html

# Regenerates internal/gtk from the GIR files in the stand (tooling/girgen) and checks the generated
# struct layouts against the C compiler.
gtk-gen: linux-vnc
	docker exec $(LINUX_CTR) sh -c 'cd /src && go run ./tooling/girgen && sh tooling/girgen/verify.sh'

linux-shell: linux-vnc
	docker exec -it $(LINUX_CTR) bash

linux-run: linux-vnc
	docker exec $(LINUX_CTR) sh -c '$(CMD)'
