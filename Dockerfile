FROM golang:1.21.9-alpine3.19 AS builder

WORKDIR /go/src/

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN go build -trimpath -o /go/bin/tracking-bars cmd/main/main.go

FROM alpine:3.19

RUN apk update && \
  apk add --no-cache \
  curl \
  mtr \
  traceroute \
  mailcap

COPY --from=builder /go/bin/tracking-bars /usr/local/bin/tracking-bars
RUN chmod +x /usr/local/bin/tracking-bars

CMD ["/usr/local/bin/tracking-bars"]
