## version, taken from Git tag (like v1.0.0) or hash
VER := $(shell git describe --always --dirty 2>/dev/null | sed -e 's/^v//g' )

## fully-qualified path to this Makefile
MKFILE_PATH := $(realpath $(lastword $(MAKEFILE_LIST)))

## fully-qualified path to the current directory
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

## all non-test source files
SOURCES := go.mod go.sum $(shell go list -f '{{range .GoFiles}}{{ $$.Dir }}/{{.}} {{end}}' ./... | sed -e 's@$(CURRENT_DIR)/@@g' )

TEST_SOURCES := go.mod go.sum $(shell go list -f '{{range .XTestGoFiles}}{{ $$.Dir }}/{{.}} {{end}}' ./... | sed -e 's@$(CURRENT_DIR)/@@g' )

.PHONY: all
all: build

.PHONY: clean
clean:
	git clean -f -Xd

## ensure we're using the version of ginkgo specified in go.mod
GINKGO := $(shell awk '/github.com\/onsi\/ginkgo v/ {printf("stage/ginkgo@%s", $$2)}' go.mod)
$(GINKGO):
	go build -o $@ github.com/onsi/ginkgo/ginkgo

.PHONY: tools
tools: $(GINKGO)

stage/.tests-ran: $(SOURCES) $(TEST_SOURCES) $(GINKGO)
	@$(GINKGO) -r
	@touch $@

.PHONY: test
test: stage/.tests-ran

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
