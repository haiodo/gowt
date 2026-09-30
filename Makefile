export CGO_ENABLED := 0

SWT_REPO ?= $(HOME)/Develop/repos/eclipse.platform.swt
export SWT_REPO

CMDS := $(notdir $(wildcard cmd/*))
BIN  := bin

.PHONY: all gen build release release-sizes-update vet test check clean test-swt test-swt-update snap-check snap-update run-% $(CMDS)

all: check build

# Rebuilds the translator and regenerates swt/, internal/cocoa/ and examples/ from the SWT sources.
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

check: vet test

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
