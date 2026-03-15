# AGENTS.md

## Cursor Cloud specific instructions

### Overview

Imuachain is a single Go binary (`imuad`) — an omnichain restaking protocol built on Cosmos SDK v0.47.8, CometBFT v0.37.4, and Evmos v16 (EVM-compatible). There is no frontend, no external database, and no microservices.

### Prerequisites (pre-installed in snapshot)

- **Go 1.21.12** at `/usr/local/go/bin/go` — the project requires this exact minor version (set in `go.mod` and `.golangci.yml`). Ensure `PATH` includes `/usr/local/go/bin`.
- **GCC** — required for CGO (used by the build).
- **golangci-lint v1.59.1** — matches CI. Installed at `/usr/local/bin/golangci-lint`.

### Key commands

| Task | Command | Notes |
|------|---------|-------|
| Build | `make build` | Produces `./build/imuad` and `./build/imuachain-test-tool` |
| Unit tests | `make test` | Runs `go test` with `--tags devmode`, ~2 min |
| Lint (Go) | `golangci-lint run --timeout 10m --out-format=tab` | CI uses `--timeout 10m` |
| Format | `make format-golang` | Requires `gofumpt` |
| Install | `make install` | Installs `imuad` to `$GOPATH/bin` |

### Gotchas

- **CGO is mandatory**: All `go build`/`go test` commands must run with `CGO_ENABLED=1 CGO_CFLAGS="-std=gnu11"`. The Makefile handles this automatically.
- **Linker warnings** (`missing .note.GNU-stack section`) during build/test are harmless and expected.
- **`make lint`** also runs `solhint` on Solidity contracts (requires Node.js + solhint). For Go-only lint, call `golangci-lint` directly.
- **Local node startup** (`local_node.sh`) requires `ALCHEMY_API_KEY`, `BOOTSTRAP` env vars, and Foundry's `cast` CLI. It sets up complex genesis state for operator/delegation/oracle modules.
- **`testnet init-files`** creates genesis that lacks operator registrations needed by the `x/dogfood` module, so nodes will panic on start. Use `local_node.sh` for a working single-node setup.
- **Protobuf generation** (`make proto-gen`) runs inside Docker and is only needed when `.proto` files change.
- **Default base branch** is `develop` (see `.cursor/rules/git-conventions.mdc`).
