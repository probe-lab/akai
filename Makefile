GOCC=go

GIT_PACKAGE=github.com/probe-lab/akai
TARGET_PATH=./cmd/akai
BIN_PATH=./build
BIN=./build/akai

.PHONY: install build clean


install:
	$(GOCC) install $(GIT_PACKAGE)

uninstall:
	$(GOCC) clean $(GIT_PACKAGE)

build:
	$(GOCC) get $(TARGET_PATH)
	$(GOCC) build -o $(BIN) $(TARGET_PATH)

clean:
	rm -r $(BIN_PATH)