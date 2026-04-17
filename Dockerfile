# This is the published docker image for imuachain.

FROM golang:1.23.11-alpine3.21 AS build-env

WORKDIR /go/src/github.com/imua-xyz/imuachain

COPY go.mod go.sum ./

# Rely on the pinned Alpine 3.21 base image for reproducibility; fuzzy apk
# pins break CI whenever the stable index advances within a release line.
# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates build-base git linux-headers

RUN --mount=type=bind,target=. --mount=type=secret,id=GITHUB_TOKEN \
    git config --global url."https://$(cat /run/secrets/GITHUB_TOKEN)@github.com/".insteadOf "https://github.com/"; \
    go mod download

COPY . .

RUN make build && go install github.com/MinseokOh/toml-cli@latest

FROM alpine:3.21

WORKDIR /root

COPY --from=build-env /go/src/github.com/imua-xyz/imuachain/build/imuad /usr/bin/imuad
COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli

# Rely on the pinned Alpine 3.21 base image for reproducibility; fuzzy apk
# pins break CI whenever the stable index advances within a release line.
# hadolint ignore=DL3018
RUN apk add --no-cache \
	ca-certificates \
	libstdc++ \
	jq \
	curl \
	bash \
    && addgroup -g 1000 imua \
    && adduser -S -h /home/imua -D imua -u 1000 -G imua

USER 1000
WORKDIR /home/imua

EXPOSE 26656 26657 1317 9090 8545 8546

# Every 30s, allow 3 retries before failing, timeout after 30s.
HEALTHCHECK --interval=30s --timeout=30s --retries=3 CMD curl -f http://localhost:26657/health || exit 1

CMD ["imuad"]
