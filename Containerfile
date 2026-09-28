# -----------------------------------------------------------------------------
# Build stage
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/goweeb-cli \
    ./cmd/goweeb-cli

RUN CGO_ENABLED=0 \
    go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /out/goweeb-tui \
    ./cmd/goweeb-tui
