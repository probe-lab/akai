

GOCC := "go"
TARGET_PATH := "./cmd/akai"
BIN_PATH := "./build"
BIN := "./build/akai"

GIT_PACKAGE := "github.com/probe-lab/akai"
COMMIT := `git rev-parse --short HEAD`
DATE := `date "+%Y-%m-%dT%H:%M:%SZ"`
USER := `id -un`
VERSION := `git describe --tags --abbrev=0 || true`

default:
	@just --list --jusfile {{ justfile() }}
	
install: 
	{{GOCC}} install {GIT_PACKAGE}

uninstall:
	{{GOCC}} clean {{GIT_PACKAGE}}

build:
	{{GOCC}} build -o {{BIN}} {{TARGET_PATH}}

clean:
	@rm -r $(BIN_PATH)

format:
	{{GOCC}} fmt ./...
	{{GOCC}} mod tidy -v

lint:
	{{GOCC}} mod verify
	{{GOCC}} vet ./...
	{{GOCC}} run honnef.co/go/tools/cmd/staticcheck@latest ./...
	{{GOCC}} test -race -buildvcs -vet=off $(TARGET_PATH)

docker:
	docker build -t probe-lab/akai:latest -t probe-lab/akai-$(GIT_SHA) .

test:
	{{GOCC}} test -v ./core
	{{GOCC}} test -v ./avail
	{{GOCC}} test -v ./api

test-db:
	@echo "go test ./db/clickhouse"; \
	{{GOCC}} test -v ./db/clickhouse || @echo "the test requires a clickhouse db"

test-avail-api:
	@echo "go test ./avail/api"; \
	{{GOCC}} test -v ./avail/api || @echo "the test requires an avail-light client running an http-api at the 5000 port"



