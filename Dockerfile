FROM golang:1.19.4-buster

WORKDIR /go/src/

COPY . /go/src/

RUN apt-get update
RUN apt-get -y install postgresql-client

RUN sed -i -e 's/\r$//' *.sh
RUN chmod +x wait-for-postgres.sh

RUN go clean --modcache
RUN go mod download
RUN go build -o app cmd/main/main.go

CMD ["./app"]