export CGO_ENABLED := 0

SWT_REPO ?= $(HOME)/Develop/repos/eclipse.platform.swt
export SWT_REPO

# Platform port.sh translates: cocoa now, win32 and gtk are prepared slots.
PLATFORM ?= cocoa
export PLATFORM

CMDS := $(notdir $(wildcard cmd/*))
BIN  := bin

.PHONY: win-build win-probe win-hello win-swttest win-swttest-update all gen build release release-sizes-update vet test check xcheck api-check clean test-swt test-swt-update snap-check snap-update run-% $(CMDS)

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

# Records the current bin/release sizes as the reference for the 5% growth warning.
release-sizes-update: release
	@bash tooling/release-sizes.sh -update

vet:
	go vet ./...

test:
	go test ./...

check: vet test xcheck api-check

# Platform-neutral code must build for every OS and must not import a platform's PI package; the
# platform-specific rest (swt, cmd, ...) builds only where that platform's port exists.
XCHECK_PKGS := ./internal/jrt ./internal/junit ./internal/snapcmp ./cmd/snapcheck ./tooling/apidump ./examples/controlexample/res
xcheck:
	@for os in windows linux; do GOOS=$$os go build $(XCHECK_PKGS) || exit 1; done
	@bad=$$(grep -lE '"github.com/haiodo/gowt/internal/(cocoa|win32|gtk)"' $$(ls swt/*.go examples/*/*.go tests/swttests/*.go cmd/*/*.go | grep -vE '_(darwin|windows|linux)(_test)?\.go$$') || true); \
	if [ -n "$$bad" ]; then echo "platform import in files without a GOOS suffix:"; echo "$$bad"; exit 1; fi

# Exported swt API of the platforms in tooling/apidump/platforms.txt must agree (platform-only.txt lists the exceptions).
api-check:
	go run ./tooling/apidump -check

# Translated SWT JUnit tests on the main thread (cmd/swttest), gated by tests/expected.txt: fails on a
# regression or an undescribed failure. SWTTEST_FLAGS e.g. -run GC (the gate then checks only those).
test-swt: swttest
	./$(BIN)/swttest -expected tests/expected.txt $(SWTTEST_FLAGS)

# Rewrites tests/expected.txt from a full run; a new failure comes out as UNDESCRIBED until its cause is written.
test-swt-update: swttest
	./$(BIN)/swttest -update tests/expected.txt $(SWTTEST_FLAGS)

# ControlExample tab snapshots against tests/snapshots (cmd/snapcheck: per-pixel threshold + allowed
# fraction). Skips with a message when scale / macOS version / appearance differ from the references.
# Failures leave diff images in bin/snap/diff.
snap-check: controlexample snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	./$(BIN)/controlexample -snap $(BIN)/snap/got > $(BIN)/snap/run.log
	./$(BIN)/snapcheck -ref tests/snapshots -got $(BIN)/snap/got -diff $(BIN)/snap/diff

# Rewrites tests/snapshots (PNGs + meta.txt) from a fresh run.
snap-update: controlexample snapcheck
	@rm -rf $(BIN)/snap && mkdir -p $(BIN)/snap
	./$(BIN)/controlexample -snap $(BIN)/snap/got > $(BIN)/snap/run.log
	./$(BIN)/snapcheck -update -ref tests/snapshots -got $(BIN)/snap/got

run-%: %
	./$(BIN)/$*

clean:
	rm -rf $(BIN) tooling/j2go/target

# Windows port (CrossOver bottle "gowt"): cross-build the exes, run them under Wine. win-probe opens no window;
# the others do. Paths are absolute (Wine reads the host file system as drive Z:).
WINE ?= /Applications/CrossOver.app/Contents/SharedSupport/CrossOver/bin/wine
WINBOTTLE ?= gowt
WINBIN := $(BIN)/windows
WINCMDS := hello swttest controlexample winprobe

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

linux-shell: linux-vnc
	docker exec -it $(LINUX_CTR) bash

linux-run: linux-vnc
	docker exec $(LINUX_CTR) sh -c '$(CMD)'
