export CGO_ENABLED := 0

SWT_REPO ?= $(HOME)/Develop/repos/eclipse.platform.swt
export SWT_REPO

CMDS := $(notdir $(wildcard cmd/*))
BIN  := bin

.PHONY: all gen build release vet test check clean test-swt test-swt-update run-% $(CMDS)

all: check build

# Rebuilds the translator and regenerates swt/, internal/cocoa/ and examples/ from the SWT sources.
gen:
	bash tooling/port.sh

build: $(CMDS)

$(CMDS):
	go build -o $(BIN)/$@ ./cmd/$@

# Stripped builds into bin/release/ with a size per binary (pclntab stays: stack traces).
release:
	@mkdir -p $(BIN)/release
	@for c in $(CMDS); do \
		go build -trimpath -ldflags='-s -w' -o $(BIN)/release/$$c ./cmd/$$c || exit 1; \
		printf '%10d  %s\n' $$(stat -f %z $(BIN)/release/$$c) $$c; \
	done

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

run-%: %
	./$(BIN)/$*

clean:
	rm -rf $(BIN) tooling/j2go/target
