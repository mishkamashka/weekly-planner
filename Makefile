BINARY   := bin/bot
LINUX    := bin/bot-linux
CMD      := ./cmd/bot

.PHONY: run build build-linux deploy tidy clean help

## run: start locally in sandbox mode (no token or DB required)
run:
	SANDBOX=true go run $(CMD)

## build: compile binary for the current OS
build:
	mkdir -p bin
	go build -o $(BINARY) $(CMD)

## build-linux: cross-compile for Linux amd64 (for server deploy)
build-linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o $(LINUX) $(CMD)

## deploy: build for Linux and scp to the server, then restart the service
##   usage: make deploy SERVER=user@your-server-ip
deploy: build-linux
	@if [ -z "$(SERVER)" ]; then echo "usage: make deploy SERVER=user@host"; exit 1; fi
	scp $(LINUX) $(SERVER):/tmp/bot-new
	ssh $(SERVER) "mv /tmp/bot-new /usr/local/bin/bot && systemctl restart weekly-planner"

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf bin/

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## //'
