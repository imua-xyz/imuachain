#!/usr/bin/env bash

KEYS[0]="dev0"
KEYS[1]="dev1"
KEYS[2]="dev2"
# TODO: exocore testnet chainid is still under consideration and need to be finalized later
CHAINID="exocoretestnet_233-1"
MONIKER="localtestnet"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
KEYRING="test"
ALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the exocored instance
HOMEDIR="$HOME/.tmp-exocored"
# to trace evm
#TRACE="--trace"
TRACE=""

# make the validator consensus key consistent 0xf0f6919e522c5b97db2c8255bff743f9dfddd7ad9fc37cb0c1670b480d0f9914
CONSENSUS_KEY_MNEMONIC="wonder quality resource ketchup occur stadium vicious output situate plug second monkey harbor vanish then myself primary feed earth story real soccer shove like"
# the account below acts as both initial operator and local consistent faucet.
# pk: D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE
# 0x3e108c058e8066DA635321Dc3018294cA82ddEdf == exo18cggcpvwspnd5c6ny8wrqxpffj5zmhklprtnph
LOCAL_MNEMONIC="knock benefit magnet slogan normal broken frequent level video focus spell utility"
LOCAL_NAME="local_funded_account"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json
ORACLE_ENV_CHAINLINK=$HOMEDIR/config/oracle_env_chainlink.yaml
ORACLE_FEEDER=$HOMEDIR/config/oracle_feeder.yaml
ORACLE_ENV_BEACONCHAIN=$HOMEDIR/config/oracle_env_beaconchain.yaml

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}
command -v cast >/dev/null 2>&1 || {
	echo >&2 "cast not installed. More info: https://getfoundry.sh"
	exit 1
}

# used to exit on first error (any non-zero exit code)
set -e

# Reinstall daemon
make install

# User prompt if an existing local node configuration is found.
if [ -d "$HOMEDIR" ]; then
	printf "\nAn existing folder at '%s' was found. You can choose to delete this folder and start a new local node with new keys from genesis. When declined, the existing local node is started. \n" "$HOMEDIR"
	echo "Overwrite the existing configuration and start a new local node? [y/n]"
	read -r overwrite
else
	overwrite="Y"
fi

# Setup local node if overwrite is set to Yes, otherwise skip setup
if [[ $overwrite == "y" || $overwrite == "Y" ]]; then
	# Remove the previous folder
	rm -rf "$HOMEDIR"

	# Set client config
	exocored config keyring-backend $KEYRING --home "$HOMEDIR"
	exocored config chain-id $CHAINID --home "$HOMEDIR"

	# If keys exist they should be deleted
	for KEY in "${KEYS[@]}"; do
		exocored keys add "$KEY" --keyring-backend "$KEYRING" --algo $ALGO --home "$HOMEDIR"
	done

	# Use recover so that there is always a consistent address funded in the localnet genesis.
	echo "${LOCAL_MNEMONIC}" | exocored --home "$HOMEDIR" --keyring-backend "$KEYRING" keys add "${LOCAL_NAME}" --recover

	# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
	# Use recover to use a consistent consensus key for validator.
	echo "${CONSENSUS_KEY_MNEMONIC}" | exocored init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR" --recover

	# these values are derived instead of hardcoded, so that edits to mnemonic or chain-id are automatically reflected
	CHAINID_WITHOUT_REVISION=${CHAINID%-*}
	AVS_ADDRESS=0x$(cast keccak "chain-id-prefix""$CHAINID_WITHOUT_REVISION" | cast 2b | tail -c 41)
	LOCAL_ADDRESS_EXO=$(exocored keys show "$LOCAL_NAME" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
	LOCAL_ADDRESS_HEX=0x$(exocored keys parse "$LOCAL_ADDRESS_EXO" --output json | jq -r .bytes | tr '[:upper:]' '[:lower:]')
	CONSENSUS_KEY=$(exocored keys consensus-pubkey-to-bytes --keyring-backend "$KEYRING" --home "$HOMEDIR" --output json | jq -r .bytes32)

	echo "the local operator address is $LOCAL_ADDRESS_EXO"
	echo "the dogfood AVS address is $AVS_ADDRESS"

	# Change parameter token denominations to hua
	jq '.app_state["crisis"]["constant_fee"]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# When upgrade to cosmos-sdk v0.47, use gov.params to edit the deposit params
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/assets
	# Using the local funding address as the Exocore gateway address to facilitate testing for precompiles without depending on the gateway contract.
	jq '.app_state["assets"]["params"]["exocore_lz_app_address"]="'"$LOCAL_ADDRESS_HEX"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][0]["name"]="Example EVM chain"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][0]["meta_info"]="Example EVM chain meta info"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][0]["layer_zero_chain_id"]="101"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][0]["address_length"]="20"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][0]["asset_basic_info"]["name"]="Tether USD"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][0]["asset_basic_info"]["meta_info"]="Tether USD token"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][0]["asset_basic_info"]["address"]="0xdac17f958d2ee523a2206206994597c13d831ec7"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][0]["asset_basic_info"]["layer_zero_chain_id"]="101"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][0]["staking_total_amount"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["deposits"][0]["staker"]="'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["total_deposit_amount"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["withdrawable_amount"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["operator"]="'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["total_amount"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["total_share"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["operator_share"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/oracle
	jq '.app_state["oracle"]["params"]["tokens"][1]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["start_base_block"]="20"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/operator
	jq '.app_state["operator"]["operators"][0]["operator_address"]="'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["operator_info"]["earnings_addr"]="'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["operator_info"]["operator_meta_info"]="operator1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["operator_info"]["commission"]["commission_rates"]["rate"]="0.0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["operator_info"]["commission"]["commission_rates"]["max_rate"]="0.0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["operator_info"]["commission"]["commission_rates"]["max_change_rate"]="0.0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["operator_address"]="'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["chains"][0]["chain_id"]="'"$CHAINID_WITHOUT_REVISION"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["chains"][0]["consensus_key"]="'"$CONSENSUS_KEY"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["key"]="'"$LOCAL_ADDRESS_EXO"'/'"$AVS_ADDRESS"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["opted_in_height"]=1' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["opted_out_height"]="'"$(echo '2^64-1' | bc)"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["jailed"]=false' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["avs_usd_values"][0]["avs_addr"]="'"$AVS_ADDRESS"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["avs_usd_values"][0]["value"]["amount"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["key"]="'"$AVS_ADDRESS"'/'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["self_usd_value"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["total_usd_value"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["active_usd_value"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/delegation
	jq '.app_state["delegation"]["delegation_states"][0]["key"]="'"$LOCAL_ADDRESS_HEX"'_0x65/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65/'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][0]["states"]["undelegatable_share"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][0]["states"]["wait_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["associations"][0]["staker_id"]="'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["associations"][0]["operator"]="'"$LOCAL_ADDRESS_EXO"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["stakers_by_operator"][0]["key"]="'"$LOCAL_ADDRESS_EXO"'/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["stakers_by_operator"][0]["stakers"][0]="'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/dogfood
	# for easy testing, use an epoch of 1 minute and 5 epochs until unbonded.
	# i did not use 1 epoch to allow for testing that it does not happen at each epoch.
	jq '.app_state["dogfood"]["params"]["epoch_identifier"]="minute"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["epochs_until_unbonded"]="5"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["val_set"][0]["public_key"]="'"$CONSENSUS_KEY"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["val_set"][0]["power"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["last_total_power"]="5000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["min_self_delegation"]="100"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/exomint
	# set the epoch identifier to `minute`. the default set by the module is `day`,
	# which is more suitable for a live network and not a localnet.
	jq '.app_state["exomint"]["params"]["epoch_identifier"]="minute"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/oracle
	# set the asset_id to default asset
	jq '.app_state["oracle"]["params"]["tokens"][1]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["decimal"]="8"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# custom epoch definitions can be added here, if required.
	# see https://github.com/ExocoreNetwork/exocore/blob/82b2509ad33ab7679592dcb1aa56a7a811128410/local_node.sh#L123 as an example

	# generate oracle_env_chainlink.yaml file
	oracle_env_chainlink_content=$(
		cat <<EOF
urls:
  mainnet: !!str https://rpc.ankr.com/eth
  sepolia: !!str https://rpc.ankr.com/eth_sepolia
tokens:
  ETHUSDT: 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419_mainnet
  AAVEUSDT: 0x547a514d5e3769680Ce22B2361c10Ea13619e8a9_mainnet
  WSTETHUSDT: 0xaaabb530434B0EeAAc9A42E25dbC6A22D7bE218E_sepolia
EOF
	)

	# Write the YAML content to a file
	echo "$oracle_env_chainlink_content" >"$ORACLE_ENV_CHAINLINK"

	# generate oracle_feeder.yaml file
	oracle_feeder_content=$(
		cat <<EOF
sources:
  - chainlink
tokens:
  - ETHUSDT
  - WSTETHUSDT
sender:
  mnemonic: ""
  path: $HOMEDIR/config
exocore:
  chainid: $CHAINID
  appName: exocore
  rpc: 127.0.0.1:9090
  ws:
    addr: !!str ws://127.0.0.1:26657
    endpoint: /websocket
EOF
	)

	# Write the YAML content to a file
	echo "$oracle_feeder_content" >"$ORACLE_FEEDER"

	# generate oracle_env_beaconchain.yaml
	oracle_env_beaconchain_content=$(
		cat <<EOF
url:
  !!str https://ethereum-holesky-rpc.publicnode.com
EOF
	)

	# Write the YAML content to a file
	echo "$oracle_env_beaconchain_content" >"$ORACLE_ENV_BEACONCHAIN"

	if [[ $1 == "pending" ]]; then
		if [[ "$OSTYPE" == "darwin"* ]]; then
			sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
			sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
			sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
			sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
			sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
		else
			sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
			sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
			sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
			sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
			sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
		fi
	fi

	# remove evmos seeds for localnet
	if [[ "$OSTYPE" == "darwin"* ]]; then
		sed -i '' 's/seeds = "[^"]*"/seeds = ""/' "$CONFIG"
	else
		sed -i 's/seeds = "[^"]*"/seeds = ""/' "$CONFIG"
	fi

	# enable prometheus metrics
	if [[ "$OSTYPE" == "darwin"* ]]; then
		sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
		sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
		sed -i '' 's/enabled = false/enabled = true/g' "$APP_TOML"
		sed -i '' 's/enable = false/enable = true/g' "$APP_TOML"
	else
		sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
		sed -i 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
		sed -i 's/enabled = false/enabled = true/g' "$APP_TOML"
	fi

	# Change proposal periods to pass within a reasonable time for local testing
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$HOMEDIR"/config/genesis.json
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$HOMEDIR"/config/genesis.json

	# set custom pruning settings
	sed -i.bak 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
	sed -i.bak 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
	sed -i.bak 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"

	# make sure the localhost IP is 0.0.0.0
	sed -i.bak 's/localhost/0.0.0.0/g' "$CONFIG"
	sed -i.bak 's/localhost/0.0.0.0/g' "$APP_TOML"
	sed -i.bak 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

	# Allocate genesis accounts (cosmos formatted addresses)
	for KEY in "${KEYS[@]}"; do
		exocored add-genesis-account "$KEY" 100000000000000000000000000hua --keyring-backend "$KEYRING" --home "$HOMEDIR"
	done
	exocored add-genesis-account "${LOCAL_NAME}" 100000000000000000000000000hua --keyring-backend "$KEYRING" --home "$HOMEDIR"

	# bc is required to add these big numbers
	# note the extra +1 is for LOCAL_NAME
	total_supply=$(echo "(${#KEYS[@]} + 1) * 100000000000000000000000000" | bc)
	jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	exocored validate-genesis --home "$HOMEDIR"

	if [[ $1 == "pending" ]]; then
		echo "pending mode is on, please wait for the first block committed."
	fi
fi

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
exocored start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001hua --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --json-rpc.enable true --home "$HOMEDIR" --chain-id "$CHAINID" --oracle --grpc.enable true
