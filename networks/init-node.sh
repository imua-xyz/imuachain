#!/bin/bash
current_dir="$PWD"
CHAINDIR="$current_dir/build/.testnets"

for node in {0..3}; do
	NODE_DIR="$CHAINDIR/node$node"
	GENESIS="$NODE_DIR/imuad/config/genesis.json"
	TMP_GENESIS="$NODE_DIR/imuad/config/tmp_genesis.json"
	APP_TOML="$NODE_DIR/imuad/config/app.toml"
	CONFIG_TOML="$NODE_DIR/imuad/config/config.toml"
	# If TMP_GENESIS directory does not exist, create it
	TMP_GENESIS_DIR=$(dirname "$TMP_GENESIS")
	if [ ! -d "$TMP_GENESIS_DIR" ]; then
		mkdir -p "$TMP_GENESIS_DIR"
	fi
	# used to exit on first error (any non-zero exit code)
	set -e

	jq '.consensus_params["block"]["max_gas"]="1000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# make sure the localhost IP is 0.0.0.0
	sed -i.bak 's/localhost/0.0.0.0/g' "$CONFIG_TOML"
	sed -i.bak 's/localhost/0.0.0.0/g' "$APP_TOML"
	sed -i.bak 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

	# enable prometheus metrics
	sed -i.bak 's/prometheus = false/prometheus = true/' "$CONFIG_TOML"
	sed -i.bak 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
	sed -i.bak 's/enabled = false/enabled = true/g' "$APP_TOML"

	# use timeout_commit 1s to make test faster
	sed -i.bak 's/timeout_commit = "3s"/timeout_commit = "1s"/g' "$CONFIG_TOML"

	# Enable the APIs for the tests to be successful
	sed -i.bak 's/enable = false/enable = true/g' "$APP_TOML"
	sed -i.bak 's/swagger = false/swagger = true/g' "$APP_TOML"
	sed -i.bak 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' "$APP_TOML"

	# remove seeds
	sed -i.bak 's/seeds = "[^"]*"/seeds = ""/' "$CONFIG_TOML"

	echo "Modified configurations for node$node"
done
