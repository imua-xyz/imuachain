#!/bin/bash

KEY="dev0"
CHAINID="imuachainlocalnet_232-1"
MONIKER="mymoniker"
DATA_DIR=$(mktemp -d -t imua-datadir.XXXXX)

echo "create and add new keys"
./imuad keys add "${KEY}" --home "${DATA_DIR}" --no-backup --chain-id "${CHAINID}" --algo "eth_secp256k1" --keyring-backend test
echo "init imua with moniker=\"${MONIKER}\" and chain-id=\"${CHAINID}\""
./imuad init "${MONIKER}" --chain-id "${CHAINID}" --home "${DATA_DIR}"
echo "prepare genesis: Allocate genesis accounts"
./imuad add-genesis-account \
	"$(./imuad keys show "${KEY}" -a --home "${DATA_DIR}" --keyring-backend test)" 1000000000000000000aevmos,1000000000000000000stake \
	--home "${DATA_DIR}" --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
./imuad gentx "${KEY}" 1000000000000000000stake --home "${DATA_DIR}" --keyring-backend test --chain-id "${CHAINID}"
echo "prepare genesis: Collect genesis tx"
./imuad collect-gentxs --home "${DATA_DIR}"
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./imuad validate-genesis --home "${DATA_DIR}"

echo "starting imua node in background ..."
./imuad start --pruning=nothing --rpc.unsafe \
	--keyring-backend test --home "${DATA_DIR}" \
	>"${DATA_DIR}"/node.log 2>&1 &
disown

echo "started imua node"
tail -f /dev/null
