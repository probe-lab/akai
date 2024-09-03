FROM golang:1.22 AS builder

# Switch to an isolated build directory
WORKDIR /build

# For caching, only copy the dependency-defining files and download depdencies
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o akai ./cmd/akai

RUN ls -l .

# Create lightweight image. Use Debian as base because we have enabled CGO
# above and hence compile against glibc. This means we can't use alpine.
FROM debian:latest

# Create user akai
RUN adduser --system --no-create-home --disabled-login --group akai
WORKDIR /home/akai
USER akai

COPY --from=builder /build/akai ./akai
COPY --from=builder /build/db/clickhouse/migrations ./db/clickhouse/migrations

CMD ["./akai"]