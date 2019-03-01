## the publicly-visible name of your binary; defaults to name of directory
NAME = $(notdir $(CURRENT_DIR))

## version, taken from Git tag (like v1.0.0) or hash
VER := $(shell (git describe --always --dirty 2>/dev/null || echo '<unknown>') | sed -e 's/^v//g' )

## fully-qualified path to this Makefile
MKFILE_PATH := $(realpath $(lastword $(MAKEFILE_LIST)))
## fully-qualified path to the current directory
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

BIN = .godeps/bin

GO_MOD_SOURCES := go.mod go.sum
SOURCES := $(shell go list -f '{{range .GoFiles}}{{ $$.Dir }}/{{.}} {{end}}' ./... | sed -e 's@$(CURRENT_DIR)/@@g' )
TEST_SOURCES := $(shell go list -f '{{range .TestGoFiles}}{{ $$.Dir }}/{{.}} {{end}} {{range .XTestGoFiles}}{{ $$.Dir }}/{{.}} {{end}} ' ./... | sed -e 's@$(CURRENT_DIR)/@@g')

## targets after a | are order-only; the presence of the target is sufficient
## http://stackoverflow.com/questions/4248300/in-a-makefile-is-a-directory-name-a-phony-target-or-real-target

.PHONY: all
all: build

.PHONY: clean
clean:
	git clean -f -Xd

$(BIN) stage:
	@mkdir -p $@

$(BIN)/ginkgo: $(GO_MOD_SOURCES) | $(BIN)
	go build -o $@ github.com/onsi/ginkgo/ginkgo
	@touch $@

$(BIN)/mockery: $(GO_MOD_SOURCES) | $(BIN)
	go build -o $@ github.com/vektra/mockery
	@touch $@

## installs dev tools
.PHONY: devtools
devtools: $(BIN)/ginkgo $(BIN)/mockery

## run tests
stage/.tests_ran: $(GO_MOD_SOURCES) $(TEST_SOURCES) $(SOURCES) $(BIN)/ginkgo | stage
	$(BIN)/ginkgo -r
	@touch $@

.PHONY: test
test: stage/.tests_ran

.PHONY: watch-tests
watch-tests: $(GINKGO)
	@$(GINKGO) watch -r

.PHONY: build
build: stage/nomad-watcher stage/nomad-tail
stage/nomad-watcher stage/nomad-tail: $(SOURCES) | test
	go build -o $@ -ldflags '-X main.version=$(VER)' ./cmd/$(notdir $@)

.PHONY: linux
linux: stage/nomad-watcher-linux stage/nomad-tail-linux
stage/nomad-watcher-linux stage/nomad-tail-linux: $(SOURCES) | test
	GOOS=linux GOARCH=amd64 go build -o $@ -ldflags '-X main.version=$(VER)' ./cmd/$(patsubst %-linux,%,$(notdir $@))

.PHONY: darwin
darwin: stage/nomad-watcher-darwin stage/nomad-tail-darwin
stage/nomad-watcher-darwin stage/nomad-tail-darwin: $(SOURCES) | test
	GOOS=darwin GOARCH=amd64 go build -o $@ -ldflags '-X main.version=$(VER)' ./cmd/$(patsubst %-darwin,%,$(notdir $@))
