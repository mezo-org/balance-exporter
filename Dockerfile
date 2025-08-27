FROM golang:1.18-alpine AS base

RUN apk add --update --no-cache \
    g++

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN go mod tidy
RUN go build -o balance-exporter

FROM alpine as runtime

ENV BIN_PATH=/usr/local/bin

COPY --from=base /app/balance-exporter $BIN_PATH

ENV CHAIN_RPC_URL https://mainnet.infura.io
ENV PORT 9015
ENV ADDRESSES_FILE "/data/addresses.txt"

EXPOSE 9015

VOLUME [ "/data" ]

ENTRYPOINT ["balance-exporter"]
