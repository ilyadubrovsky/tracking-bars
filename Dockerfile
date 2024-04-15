FROM golang:1.20-buster

WORKDIR /go/src/

COPY telegram-service /go/src/

RUN apt-get update

RUN go clean --modcache
RUN go mod download
RUN go build -o telegram-service cmd/main/main.go

CMD ["./telegram-service"]