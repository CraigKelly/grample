BASEDIR=$(CURDIR)
TOOLDIR=$(BASEDIR)/script

BINARY=grample
SOURCES := $(shell find $(BASEDIR) -name '*.go')
PACKAGES := $(shell go list ./... | grep -v vendor )
TESTED=.tested

.PHONY: build
build: $(BINARY)
$(BINARY): $(SOURCES) $(TESTED)
	go build -i ./...
	go build

.PHONY: install
install: build
	go install ./...

.PHONY: clean
clean:
	rm -f $(BINARY) debug debug.test *.out cover.html sampler.test $(TESTED)
	go clean ./...

.PHONY: lint-install
lint-install:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

.PHONY: format
format:
	@go fmt $(PACKAGES)

.PHONY: lint
lint: format
	@go vet $(PACKAGES) | $(TOOLDIR)/color.py
	@golint $(PACKAGES) | $(TOOLDIR)/color.py
	@goconst $(PACKAGES) | $(TOOLDIR)/color.py

.PHONY: test
test: $(TESTED)
$(TESTED): $(SOURCES)
	$(TOOLDIR)/test

.PHONY: cover
cover: $(SOURCES) $(TESTED)
	$(TOOLDIR)/cover

.PHONY: bench
bench: $(SOURCES) $(TESTED)
	$(TOOLDIR)/bench

.PHONY: clean
update: clean
	$(TOOLDIR)/update

.PHONY: sys-update
sys-update:
	$(TOOLDIR)/sys-update

