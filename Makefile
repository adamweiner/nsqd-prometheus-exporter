GOPATH = $(shell pwd)/go
GO = GOPATH=$(GOPATH) go

PKG = .
BIN = nsqd-prometheus-exporter

VERSION = $(shell ./$(BIN) --version | cut -d" " -f 3)

.PHONY: %

default: fmt deps test build

deploy-to-s3: build
	aws s3 cp --sse AES256 $(BIN) s3://$(AWS_S3_ARTIFACTS_BUCKET)/$(CIRCLE_PROJECT_REPONAME)/$(BIN):$(VERSION)

all: build
build: fmt deps
	$(GO) build -a -o $(BIN) $(PKG)
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
	curl -s https://raw.githubusercontent.com/pote/gpm/master/bin/gpm > gpm.sh
	chmod 755 gpm.sh
	GOPATH=$(GOPATH) ./gpm.sh
	rm gpm.sh
test-deps: deps
	$(GO) get -d -t $(PKG)
	$(GO) test -a -i $(PKG)
install:
	$(GO) install
run: all
	./$(BIN)
