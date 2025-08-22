GOCC := "go"
TARGET_PATH := "./cmd/akai"
BIN_PATH := "./build"
BIN := "./build/akai"

GIT_PACKAGE := "github.com/probe-lab/akai"
REPO_SERVER := "019120760881.dkr.ecr.us-east-1.amazonaws.com"
COMMIT := `git rev-parse --short HEAD`
DATE := `date "+%Y-%m-%dT%H:%M:%SZ"`
USER := `id -un`

default:
  @just --list --justfile {{ justfile() }}


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
	{{GOCC}} test -race -buildvcs -vet=off {{TARGET_PATH}}

docker-build:
	docker build -t probe-lab/akai:latest -t probe-lab/akai-{{COMMIT}} -t {{REPO_SERVER}}/probelab:akai-{{COMMIT}} .

docker-push:
	docker push {{REPO_SERVER}}/probelab:akai-{{COMMIT}}
	docker rmi {{REPO_SERVER}}/probelab:akai-{{COMMIT}}


test:
	{{GOCC}} test -v ./core
	{{GOCC}} test -v ./avail
	{{GOCC}} test -v ./api

test-db:
	@echo "go test ./db/clickhouse"
	{{GOCC}} test -v ./db/clickhouse

# tests the interaction between the akai-avail-api and a light client
test-avail-api:
	@echo "go test ./avail/api"
	{{GOCC}} test -v ./avail/api

# generates clickhouse migrations which work with a local docker deployment
generate-local-clickhouse-migrations:
	#!/usr/bin/env bash
	OUTDIR=db/clickhouse/migrations/chlocal
	mkdir -p $OUTDIR
	for file in $(find db/clickhouse/migrations/chcluster -maxdepth 1 -name "*.sql"); do
	  filename=$(basename $file)
	  echo "Generating $OUTDIR/$filename"

	  # The "Replicated" variants don't work with a singular clickhouse deployment
	  # We're stripping that part from the file
	  sed 's/Replicated//' $file > $OUTDIR/$filename.tmp_0

	  # Enabling the JSON type is also different in both environments
	  # allow_experimental_json_type in ClickHouse Cloud
	  # enable_json_type locally
	  sed 's/allow_experimental_json_type/enable_json_type/' $OUTDIR/$filename.tmp_0 > $OUTDIR/$filename.tmp_1

	  # Add a warning message to the top of the file
	  cat <(echo -e "-- DO NOT EDIT: This file was generated with: just generate-local-clickhouse-migrations\n") $OUTDIR/$filename.tmp_1 > $OUTDIR/$filename
	  rm $OUTDIR/$filename.tmp*
	done

start-clickhouse:
    docker-compose up -d clickhouse

stop-clickhouse:
    docker-compose stop clickhouse

restart-clickhouse:
    docker-compose restart clickhouse
