PROJECT = npt
VERSION = $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X github.com/hsn723/$(PROJECT)/cmd.version=${VERSION}"

WORKDIR = /tmp/$(PROJECT)/work
BINDIR = /tmp/$(PROJECT)/bin
YQ = $(BINDIR)/yq

PATH := $(PATH):$(BINDIR)

export PATH

all: build

.PHONY: clean
clean:
	@if [ -f $(PROJECT) ]; then rm $(PROJECT); fi

.PHONY: lint
lint:
	@if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

.PHONY: test
test:
	go test --tags=test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: $(YQ)
$(YQ): $(BINDIR)
	GOBIN=$(BINDIR) go install github.com/mikefarah/yq/v4@latest

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean
	env CGO_ENABLED=0 go build $(LDFLAGS) .

$(BINDIR):
	mkdir -p $(BINDIR)

$(WORKDIR):
	mkdir -p $(WORKDIR)
