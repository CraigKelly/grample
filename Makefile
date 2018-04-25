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
	rm -f $(BINARY) debug debug.test cover.out $(TESTED)
	go clean ./...

.PHONY: lint-install
lint-install:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

.PHONY: format
format:
	go fmt $(PACKAGES)

.PHONY: lint
lint: format
	go vet $(PACKAGES)
	golint $(PACKAGES)
	goconst $(PACKAGES)

.PHONY: test
test: $(TESTED)
$(TESTED): $(SOURCES)
	$(TOOLDIR)/test

.PHONY: cover
cover: $(SOURCES) $(VERSIONOUT)
	$(TOOLDIR)/cover

.PHONY: clean
update: clean
	$(TOOLDIR)/update

.PHONY: sys-update
sys-update:
	$(TOOLDIR)/sys-update

