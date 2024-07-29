# Compiler Variables 
GOCC=go
TARGET_PATH=./cmd/akai
BIN_PATH=./build
BIN=./build/akai

# Git Variables
GIT_PACKAGE=github.com/probe-lab/akai


# Make Operations
.PHONY: install uninstall build clean test-avail-api test-db

install:
	$(GOCC) install $(GIT_PACKAGE)

uninstall:
	$(GOCC) clean $(GIT_PACKAGE)

build:
	$(GOCC) get $(TARGET_PATH)
	$(GOCC) build -o $(BIN) $(TARGET_PATH)

clean:
	rm -r $(BIN_PATH)

test: 
	$(GOCC) test -v ./core
	$(GOCC) test -v ./avail

test-db:
	@echo "go test ./db/clickhouse"; \
	$(GOCC) test -v ./db/clickhouse || @echo "the test requires a clickhouse db"

test-avail-api:
	@echo "go test ./avail/api"; \
	$(GOCC) test -v ./avail/api || @echo "the test requires an avail-light client running an http-api at the 5000 port"

