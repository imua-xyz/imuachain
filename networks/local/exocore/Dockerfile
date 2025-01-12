FROM golang:1.21.12-alpine3.19 AS build
RUN apk add --no-cache build-base=0.5-r3 git=2.43.5-r0 linux-headers=6.5-r0
# Set working directory for the build
WORKDIR /go/work
# Add source files
COPY . ./
# build exocore
#COPY ./exocored /go/work/build
RUN LEDGER_ENABLED=false make build

#####################################
FROM alpine:3.19 AS run
RUN apk add --no-cache libstdc++=13.2.1_git20231014-r0 bash=5.2.21-r0 curl=8.11.1-r0 jq=1.7.1-r0 \
    && addgroup -g 1000 exocore \
    && adduser -S -h /home/exocore -D exocore -u 1000 -G exocore
EXPOSE 26656 26657 1317 9090 8545 8546
USER exocore
CMD ["start", "--log_format", "plain", "--chain-id", "exocorelocalnet_232-1", "--metrics", "--json-rpc.api", "eth,txpool,personal,net,debug,web3", "--api.enable", "--json-rpc.enable", "true", "--minimum-gas-prices", "0.0001hua"]
# by default, a SIGKILL is sent after 10 seconds. We need to override this to allow graceful shutdown.
STOPSIGNAL SIGTERM
VOLUME /exocore
WORKDIR /exocore

COPY ./networks/local/exocore/wrapper.sh /usr/bin/wrapper.sh
COPY --from=build /go/work/build/exocored /exocore

HEALTHCHECK --interval=30s --timeout=30s --retries=3 CMD curl -f http://localhost:26657/health || exit 1

ENTRYPOINT ["/usr/bin/wrapper.sh"]
