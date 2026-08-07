FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /warpdb ./cmd/main.go

FROM alpine:3.20
RUN adduser -D -H warpdb
COPY --from=builder /warpdb /usr/local/bin/warpdb
RUN mkdir -p /data && chown warpdb /data
USER warpdb
VOLUME /data
EXPOSE 6379
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD printf "*1\r\n\$4\r\nPING\r\n" | nc -w 2 127.0.0.1 6379 || exit 1
ENTRYPOINT ["warpdb"]
CMD ["-config", "/etc/warpdb/warpdb.json"]
