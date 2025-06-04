PROJECT = npt
VERSION = $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X github.com/hsn723/$(PROJECT)/cmd.version=${VERSION}"

BINDIR = $(shell pwd)/bin

PATH := $(PATH):$(BINDIR)

export PATH

all: build

.PHONY: clean
clean:
	@rm -rf $(BINDIR)

.PHONY: lint
lint:
	@if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

.PHONY: test
test:
	go test --tags=test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean $(BINDIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/$(PROJECT) ./cmd/$(PROJECT)/

$(BINDIR):
	@mkdir -p $(BINDIR)
