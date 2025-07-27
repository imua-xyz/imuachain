FROM golang:1.23.11-alpine3.21 AS build
RUN apk add --no-cache build-base=~0.5 git=~2.47.3 linux-headers=~6.6
# Set working directory for the build
WORKDIR /go/work
# Add source files
COPY . ./
# build imuachain
#COPY ./imuad /go/work/build
RUN LEDGER_ENABLED=false make build

#####################################
FROM alpine:3.21 AS run
RUN apk add --no-cache \
    libstdc++=~14.2.0 \
    bash=~5.2.37 \
    curl=~8.12.1 \
    jq=~1.7.1 \
 && addgroup -g 1000 imua \
 && adduser -S -h /home/imua -D imua -u 1000 -G imua
EXPOSE 26656 26657 1317 9090 8545 8546
USER imua
CMD ["start", "--log_format", "plain", "--chain-id", "imuachainlocalnet_232-1", "--metrics", "--json-rpc.api", "eth,txpool,personal,net,debug,web3", "--api.enable", "--json-rpc.enable", "true", "--minimum-gas-prices", "0.0001hua"]
# by default, a SIGKILL is sent after 10 seconds. We need to override this to allow graceful shutdown.
STOPSIGNAL SIGTERM
VOLUME /imua
WORKDIR /imua

COPY ./networks/local/imuachain/wrapper.sh /usr/bin/wrapper.sh
COPY --from=build /go/work/build/imuad /imuachain

HEALTHCHECK --interval=30s --timeout=30s --retries=3 CMD curl -f http://localhost:26657/health || exit 1

ENTRYPOINT ["/usr/bin/wrapper.sh"]
