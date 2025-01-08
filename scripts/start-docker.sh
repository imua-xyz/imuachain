#!/bin/bash

KEY="dev0"
# TODO: exocore testnet chainid is still under consideration and need to be finalized later
CHAINID="exocorelocalnet_232-1"
MONIKER="mymoniker"
DATA_DIR=$(mktemp -d -t exocore-datadir.XXXXX)

echo "create and add new keys"
./exocored keys add "${KEY}" --home "${DATA_DIR}" --no-backup --chain-id "${CHAINID}" --algo "eth_secp256k1" --keyring-backend test
echo "init exocore with moniker=\"${MONIKER}\" and chain-id=\"${CHAINID}\""
./exocored init "${MONIKER}" --chain-id "${CHAINID}" --home "${DATA_DIR}"
echo "prepare genesis: Allocate genesis accounts"
./exocored add-genesis-account \
	"$(./exocored keys show "${KEY}" -a --home "${DATA_DIR}" --keyring-backend test)" 1000000000000000000aevmos,1000000000000000000stake \
	--home "${DATA_DIR}" --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
./exocored gentx "${KEY}" 1000000000000000000stake --keyring-backend test --home "${DATA_DIR}" --keyring-backend test --chain-id "${CHAINID}"
echo "prepare genesis: Collect genesis tx"
./exocored collect-gentxs --home "${DATA_DIR}"
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./exocored validate-genesis --home "${DATA_DIR}"

echo "starting exocore node in background ..."
./exocored start --pruning=nothing --rpc.unsafe \
	--keyring-backend test --home "${DATA_DIR}" \
	>"${DATA_DIR}"/node.log 2>&1 &
disown

echo "started exocore node"
tail -f /dev/null
