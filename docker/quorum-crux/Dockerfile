FROM alpine:3.8

RUN apk update --no-cache && \
    # Update and then install dependencies
    apk add unzip db zlib wrk wget libsodium-dev go bash libpthread-stubs db-dev && \
    apk -X http://dl-cdn.alpinelinux.org/alpine/edge/testing add leveldb && \
    apk add build-base cmake boost-dev git

ENV PORT=""
ENV NODE_KEY=""
ENV CRUX_PUB=""
ENV GETH_KEY=""
ENV OWN_URL=""
ENV CRUX_PRIV=""
ENV OTHER_NODES=""
ENV GETH_RPC_PORT=""
ENV GETH_PORT=""

WORKDIR /quorum

COPY bootstrap.sh bootstrap.sh
COPY istanbul-genesis.json istanbul-genesis.json
COPY passwords.txt passwords.txt
COPY istanbul-init.sh istanbul-init.sh
COPY crux-start.sh crux-start.sh
COPY istanbul-start.sh istanbul-start.sh
COPY start.sh start.sh
COPY scripts/simpleContract.js simpleContract.js
COPY scripts/test_transaction.sh test_transaction.sh

RUN chmod +x start.sh crux-start.sh istanbul-start.sh istanbul-init.sh && \
    chmod +x test_transaction.sh && \
    chmod +x bootstrap.sh && \
    ./bootstrap.sh && \
    apk del sed make git cmake build-base gcc g++ musl-dev curl-dev boost-dev

EXPOSE 9000 21000 22000

# Entrypoint for container
ENTRYPOINT ["./start.sh"]
