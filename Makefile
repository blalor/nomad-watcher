## version, taken from Git tag (like v1.0.0) or hash
VER := $(shell (git describe --always --dirty 2>/dev/null || echo "¯\\\\\_\\(ツ\\)_/¯") | sed -e 's/^v//g' )

## fully-qualified path to this Makefile
MKFILE_PATH := $(realpath $(lastword $(MAKEFILE_LIST)))

## fully-qualified path to the current directory
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

## all non-test source files
SOURCES := $(shell go list -f '{{range .GoFiles}}{{ $$.Dir }}/{{.}} {{end}}' ./... | sed -e 's@$(CURRENT_DIR)/@@g' )

CMDS := stage/nomad-watcher stage/nomad-tail

.PHONY: all
all: $(CMDS)

.PHONY: clean
clean:
	git clean -f -Xd

$(GOPATH)/bin:
	mkdir -p $@

DEP := $(GOPATH)/bin/dep
$(DEP): | $(GOPATH)/bin
	curl -sfSL -o $@ https://github.com/golang/dep/releases/download/v0.3.2/dep-$(shell go env GOOS)-$(shell go env GOARCH)
	@chmod +x $@
	@touch $@

vendor: $(DEP) Gopkg.toml
	$(DEP) ensure

GINKGO := $(GOPATH)/bin/ginkgo
$(GINKGO): | vendor
	cd vendor/github.com/onsi/ginkgo/ginkgo && go install .

.PHONY: tools
tools: $(GINKGO)

.PHONY: test
test: $(GINKGO)
	@$(GINKGO) -r

.PHONY: watch-tests
watch-tests: $(GINKGO)
	@$(GINKGO) watch -r

$(CMDS): test
	go build -o $@ -ldflags '-X main.version=$(VER)' ./cmd/$(notdir $@)
