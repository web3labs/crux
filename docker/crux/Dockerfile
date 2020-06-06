FROM alpine:3.8

RUN apk update --no-cache && \
    # Update and then install dependencies
    apk add unzip db zlib wrk wget libsodium-dev go bash libpthread-stubs db-dev && \
    apk -X http://dl-cdn.alpinelinux.org/alpine/edge/testing add leveldb && \
    apk add build-base cmake boost-dev git

ENV CRUX_PUB=""
ENV CRUX_PRIV=""
ENV OWN_URL=""
ENV OTHER_NODES=""
ENV PORT=""

RUN git clone https://github.com/blk-io/crux.git

WORKDIR /crux

RUN make setup && \
    make build && \
    apk del sed make git cmake build-base gcc g++ musl-dev curl-dev boost-dev
# fails https://github.com/golang/go/issues/14481
# RUN make test

EXPOSE 9000

COPY start.sh start.sh
RUN chmod +x start.sh

ENTRYPOINT ["./start.sh"]