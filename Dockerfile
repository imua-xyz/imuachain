# This is the published docker image for imuachain.

FROM golang:1.23.11-alpine3.21 AS build-env

WORKDIR /go/src/github.com/imua-xyz/imuachain

COPY go.mod go.sum ./

RUN apk add --no-cache ca-certificates=~20250911 build-base=~0.5 git=~2.47.3 linux-headers=~6.6

RUN --mount=type=bind,target=. --mount=type=secret,id=GITHUB_TOKEN \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN)@github.com/".insteadOf "https://github.com/"; \
    go mod download

COPY . .

RUN make build && go install github.com/MinseokOh/toml-cli@latest

FROM alpine:3.21

WORKDIR /root

COPY --from=build-env /go/src/github.com/imua-xyz/imuachain/build/imuad /usr/bin/imuad
COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli

RUN apk add --no-cache \
	ca-certificates=~20250911 \
	libstdc++=~14.2.0 \
	jq=~1.7.1 \
	curl=~8.14.1 \
	bash=~5.2.37 \
    && addgroup -g 1000 imua \
    && adduser -S -h /home/imua -D imua -u 1000 -G imua

USER 1000
WORKDIR /home/imua

EXPOSE 26656 26657 1317 9090 8545 8546

# Every 30s, allow 3 retries before failing, timeout after 30s.
HEALTHCHECK --interval=30s --timeout=30s --retries=3 CMD curl -f http://localhost:26657/health || exit 1

CMD ["imuad"]
