FROM golang:1.20-buster

WORKDIR /go/src/

COPY . /go/src/

RUN apt-get update

RUN go clean --modcache
RUN go mod download
RUN go build -o grades-service cmd/main/main.go

CMD ["./grades-service"]