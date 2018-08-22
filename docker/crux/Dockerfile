FROM golang:1.10.3-alpine3.8 AS builder

RUN apk add --no-cache --update unzip db zlib wrk wget libsodium-dev go bash libpthread-stubs db-dev curl && \
    apk -X http://dl-cdn.alpinelinux.org/alpine/edge/testing add --no-cache leveldb && \
    apk --no-cache --update add build-base cmake boost-dev git

ENV CRUX_PUB=""
ENV CRUX_PRIV=""
ENV OWN_URL=""
ENV OTHER_NODES=""
ENV PORT=""

WORKDIR /go/src
RUN git clone https://github.com/blk-io/crux.git
WORKDIR crux
RUN make setup && \
    make build && \
    apk del sed make cmake build-base gcc g++ musl-dev curl-dev boost-dev
# fails https://github.com/golang/go/issues/14481
# RUN make test

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep

EXPOSE 9000

WORKDIR /go/src
RUN mkdir client
WORKDIR client

COPY Gopkg.toml Gopkg.toml
COPY client_test.go client_test.go

RUN dep ensure
#RUN dep ensure && \
#    go build

WORKDIR /go/src/crux

COPY start.sh start.sh
RUN chmod +x start.sh
COPY crux_start.sh crux_start.sh
RUN chmod +x crux_start.sh


ENTRYPOINT ["./start.sh"]