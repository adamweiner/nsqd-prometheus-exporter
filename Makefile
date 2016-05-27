SHELL := /bin/bash
PWD    = $(shell pwd)

GOPATH=$(PWD)/go
GO=GOPATH=$(GOPATH) go
GODEBUG=GOPATH=$(GOPATH) PATH=$(GOPATH)/bin:$$PATH godebug


PKG  = . # $(dir $(wildcard ./*)) # uncomment for implicit submodules
BIN  = nsqd-prometheus-exporter

FIND_STD_DEPS = $(GO) list std | sort | uniq
FIND_PKG_DEPS = $(GO) list -f '{{join .Deps "\n"}}' $(PKG) | sort | uniq | grep -v "^_"
DEPS          = $(shell comm -23 <($(FIND_PKG_DEPS)) <($(FIND_STD_DEPS)))
VERSION       = $(shell ./$(BIN) --version | cut -d" " -f 3)

.PHONY: %

default: fmt deps test build

deploy-to-s3: build
	aws s3 cp --sse AES256 $(BIN) s3://$(AWS_S3_ARTIFACTS_BUCKET)/$(CIRCLE_PROJECT_REPONAME)/$(BIN):$(VERSION)

all: build
build: fmt #deps
	$(GO) build -o $(BIN) $(PKG)
lint: vet
vet: deps
	$(GO) get code.google.com/p/go.tools/cmd/vet
	$(GO) vet $(PKG)
fmt:
	$(GO) fmt $(PKG)
test: test-deps
	$(GO) test $(PKG)
cover: test-deps
	$(GO) test -cover $(PKG)
clean:
	$(GO) clean -i $(PKG)
clean-all:
	$(GO) clean -i -r $(PKG)
deps:
	curl -s https://raw.githubusercontent.com/bottlenose-inc/gpm/v1.3.2/bin/gpm > gpm.sh
	chmod 755 gpm.sh
	GOPATH=$(GOPATH) ./gpm.sh
	rm gpm.sh
test-deps: deps
	$(GO) get -d -t $(PKG)
	$(GO) test -i $(PKG)
install:
	$(GO) install
run: all
	./$(BIN)
