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


# -----------------------------------------------------------------------------
# Runtime stage
# -----------------------------------------------------------------------------
FROM alpine:3.24

RUN apk add --no-cache ca-certificates \
    && addgroup -S goweeb \
    && adduser \
        -S \
        -G goweeb \
        -h /home/goweeb \
        goweeb \
    && mkdir -p /home/goweeb/Documents \
    && chown -R goweeb:goweeb /home/goweeb

COPY --from=builder /out/goweeb-cli /usr/local/bin/goweeb-cli
COPY --from=builder /out/goweeb-tui /usr/local/bin/goweeb-tui

COPY container/entrypoint.sh /usr/local/bin/goweeb

RUN chmod +x /usr/local/bin/goweeb

ENV HOME=/home/goweeb
ENV TERM=xterm-256color

USER goweeb
WORKDIR /home/goweeb

ENTRYPOINT ["/usr/local/bin/goweeb"]
CMD ["tui"]
