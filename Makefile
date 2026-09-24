export CGO_ENABLED := 0

SWT_REPO ?= $(HOME)/Develop/repos/eclipse.platform.swt
export SWT_REPO

CMDS := $(notdir $(wildcard cmd/*))
BIN  := bin

.PHONY: all gen build vet test check clean run-% $(CMDS)

all: check build

# Rebuilds the translator and regenerates swt/, internal/cocoa/ and examples/ from the SWT sources.
gen:
	bash tooling/port.sh

build: $(CMDS)

$(CMDS):
	go build -o $(BIN)/$@ ./cmd/$@

vet:
	go vet ./...

test:
	go test ./...

check: vet test

run-%: %
	./$(BIN)/$*

clean:
	rm -rf $(BIN) tooling/j2go/target
