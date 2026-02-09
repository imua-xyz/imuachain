#!/usr/bin/env bash

# check that ALCHEMY_API_KEY is set
if [ -z "$ALCHEMY_API_KEY" ]; then
	echo "ALCHEMY_API_KEY is not set"
	exit 1
fi
if [ -z "$BOOTSTRAP" ]; then
	echo "BOOTSTRAP is not set"
	exit 1
fi

KEYS[0]="dev0"
KEYS[1]="dev1"
KEYS[2]="dev2"
CHAINID="imuachainlocalnet_232-1"
MONIKER="localnet"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
KEYRING="test"
ALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the imuad instance
HOMEDIR="$HOME/.tmp-imuad"
# to trace evm
#TRACE="--trace"
TRACE=""

# make the validator consensus key consistent 0xf0f6919e522c5b97db2c8255bff743f9dfddd7ad9fc37cb0c1670b480d0f9914
CONSENSUS_KEY_MNEMONIC="wonder quality resource ketchup occur stadium vicious output situate plug second monkey harbor vanish then myself primary feed earth story real soccer shove like"
# the account below acts as both initial operator and local consistent faucet.
# pk: D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE
# 0x3e108c058e8066DA635321Dc3018294cA82ddEdf == im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj
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
ORACLE_ENV_SOLANA=$HOMEDIR/config/oracle_env_solana.yaml

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}
command -v cast >/dev/null 2>&1 || {
	echo >&2 "cast not installed. More info: https://getfoundry.sh"
	exit 1
}
command -v bc >/dev/null 2>&1 || {
	echo >&2 "bc not installed. More info: https://www.gnu.org/software/bc/manual/bc.html"
	exit 1
}

# ensure ALCHEMY_API_KEY is set
if [ -z "$ALCHEMY_API_KEY" ]; then
	echo "ALCHEMY_API_KEY is not set"
	exit 1
fi

# ensure BOOTSTRAP is set (used by oracle_env_beaconchain.yaml)
if [ -z "$BOOTSTRAP" ]; then
	echo "BOOTSTRAP is not set"
	exit 1
fi

# darwin means sed
if [[ "$OSTYPE" == "darwin"* ]]; then
	SED_CMD="sed -i ''"
else
	SED_CMD="sed -i"
fi

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
	imuad config keyring-backend $KEYRING --home "$HOMEDIR"
	imuad config chain-id $CHAINID --home "$HOMEDIR"

	# If keys exist they should be deleted
	for KEY in "${KEYS[@]}"; do
		imuad keys add "$KEY" --keyring-backend "$KEYRING" --algo $ALGO --home "$HOMEDIR"
	done

	# Use recover so that there is always a consistent address funded in the localnet genesis.
	echo "${LOCAL_MNEMONIC}" | imuad --home "$HOMEDIR" --keyring-backend "$KEYRING" keys add "${LOCAL_NAME}" --recover

	# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
	# Use recover to use a consistent consensus key for validator.
	echo "${CONSENSUS_KEY_MNEMONIC}" | imuad init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR" --recover

	# these values are derived instead of hardcoded, so that edits to mnemonic or chain-id are automatically reflected
	CHAINID_WITHOUT_REVISION=${CHAINID%-*}
	AVS_ADDRESS=0x$(cast keccak "chain-id-prefix""$CHAINID_WITHOUT_REVISION" | cast 2b | tail -c 41)
	LOCAL_ADDRESS_IM=$(imuad keys show "$LOCAL_NAME" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
	LOCAL_ADDRESS_HEX=0x$(imuad keys parse "$LOCAL_ADDRESS_IM" --output json | jq -r .bytes | tr '[:upper:]' '[:lower:]')
	CONSENSUS_KEY=$(imuad keys consensus-pubkey-to-bytes --keyring-backend "$KEYRING" --home "$HOMEDIR" --output json | jq -r .bytes32)

	echo "the local operator address is $LOCAL_ADDRESS_IM"
	echo "the dogfood AVS address is $AVS_ADDRESS"

	DEV0_ADDR=$(imuad keys show "${KEYS[0]}" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
	DEV1_ADDR=$(imuad keys show "${KEYS[1]}" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")

	POLICY_ADDR="im1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzswhsj4w"

	jq --arg local "$LOCAL_ADDRESS_IM" \
		--arg dev0 "$DEV0_ADDR" \
		--arg dev1 "$DEV1_ADDR" \
		--arg policy "$POLICY_ADDR" \
		'.app_state["group"]["groups"] = [
        {
            "id": "1",
            "admin": $policy,
            "metadata": "Genesis Admin Group",
            "version": "1",
            "total_weight": "3",
            "created_at": "0001-01-01T00:00:00Z"
        }
    ] |
    .app_state["group"]["group_members"] = [
        {
            "group_id": "1",
            "member": { "address": $dev0, "weight": "1", "metadata": "dev0", "added_at": "0001-01-01T00:00:00Z" }
        },
        {
            "group_id": "1",
            "member": { "address": $dev1, "weight": "1", "metadata": "dev1", "added_at": "0001-01-01T00:00:00Z" }
        },
        {
            "group_id": "1",
            "member": { "address": $local, "weight": "1", "metadata": "local_funded_account", "added_at": "0001-01-01T00:00:00Z" }
        }
    ] |
    .app_state["group"]["group_policies"] = [
        {
            "address": $policy,
            "group_id": "1",
            "admin": $policy,
            "metadata": "Admin Policy",
            "version": "1",
            "decision_policy": {
                "@type": "/cosmos.group.v1.ThresholdDecisionPolicy",
                "threshold": "2",
                "windows": {
                    "voting_period": "1800s",
                    "min_execution_period": "0s"
                }
            },
            "created_at": "0001-01-01T00:00:00Z"
        }
    ] |
    .app_state["group"]["group_seq"] = "1" |
    .app_state["group"]["group_policy_seq"] = "1"' \
		"$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	echo "x/group populated with Policy Address: $POLICY_ADDR"

	# Change parameter token denominations to hua
	jq '.app_state["crisis"]["constant_fee"]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# When upgrade to cosmos-sdk v0.47, use gov.params to edit the deposit params
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="hua"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/assets
	# Using the local funding address as the Imuachain gateway address to facilitate testing for precompiles without depending on the gateway contract.
	jq '.app_state["assets"]["params"]["gateways"][0]="'"$LOCAL_ADDRESS_HEX"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["params"]["gateways"][1]="0xd20f848930ef3b5eb58384d0fcd3a485533386a3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["client_chains"][1]["name"]="Example EVM chain"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][1]["meta_info"]="Example EVM chain meta info"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][1]["layer_zero_chain_id"]="101"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][1]["address_length"]="20"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["client_chains"][2]["name"]="Example solana chain"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][2]["meta_info"]="Example solana chain meta info"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][2]["layer_zero_chain_id"]="291"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["client_chains"][2]["address_length"]="20"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["tokens"][1]["asset_basic_info"]["name"]="Tether USD"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][1]["asset_basic_info"]["meta_info"]="Tether USD token"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][1]["asset_basic_info"]["address"]="0xdac17f958d2ee523a2206206994597c13d831ec7"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][1]["asset_basic_info"]["layer_zero_chain_id"]="101"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][1]["staking_total_amount"]="10000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][1]["asset_basic_info"]["decimals"]="6"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["tokens"][2]["asset_basic_info"]["name"]="nsteth"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][2]["asset_basic_info"]["decimals"]="18"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][2]["asset_basic_info"]["meta_info"]="eth native token"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][2]["asset_basic_info"]["address"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][2]["asset_basic_info"]["layer_zero_chain_id"]="101"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][2]["staking_total_amount"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["tokens"][3]["asset_basic_info"]["name"]="nstsol"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][3]["asset_basic_info"]["decimals"]="9"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][3]["asset_basic_info"]["meta_info"]="sol native token"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][3]["asset_basic_info"]["address"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][3]["asset_basic_info"]["layer_zero_chain_id"]="291"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["tokens"][3]["staking_total_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["deposits"][0]["staker"]="'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["total_deposit_amount"]="5000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["withdrawable_amount"]="1000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][0]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][1]["asset_id"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][1]["info"]["total_deposit_amount"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][1]["info"]["withdrawable_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][0]["deposits"][1]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# add another genesis staker for the tests using local node.
	jq '.app_state["assets"]["deposits"][1]["staker"]="0xdc77c5b9d061ae2f7b35a6c8854b8f5bbea98eef_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][1]["deposits"][0]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][1]["deposits"][0]["info"]["total_deposit_amount"]="5000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][1]["deposits"][0]["info"]["withdrawable_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["deposits"][1]["deposits"][0]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["assets"]["operator_assets"][0]["operator"]="'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["total_amount"]="9000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["total_share"]="9000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][0]["info"]["operator_share"]="4000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][1]["asset_id"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][1]["info"]["total_amount"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][1]["info"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][1]["info"]["total_share"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["assets"]["operator_assets"][0]["assets_state"][1]["info"]["operator_share"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/feemarket
	jq '.app_state["feemarket"]["params"]["base_fee"]="10"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/operator
	jq '.app_state["operator"]["operators"][0]["operator_addr"]="'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["description"]["moniker"]="operator1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["commission"]["commission_rates"]["rate"]="0.05"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["commission"]["commission_rates"]["max_rate"]="1.0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operators"][0]["commission"]["commission_rates"]["max_change_rate"]="0.1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["operator_address"]="'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["chains"][0]["chain_id"]="'"$CHAINID_WITHOUT_REVISION"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_records"][0]["chains"][0]["consensus_key"]="'"$CONSENSUS_KEY"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["key"]="'"$LOCAL_ADDRESS_IM"'/'"$AVS_ADDRESS"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["opted_in_height"]=1' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["opted_out_height"]="'"$(echo '2^64-1' | bc)"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["opt_states"][0]["opt_info"]["jailed"]=false' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["avs_usd_values"][0]["avs_addr"]="'"$AVS_ADDRESS"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["avs_usd_values"][0]["value"]["amount"]="12000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["key"]="'"$AVS_ADDRESS"'/'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["self_usd_value"]="7000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["total_usd_value"]="12000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_usd_values"][0]["opted_usd_value"]["active_usd_value"]="12000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_asset_usd_values"][0]["key"]="minute/'"$LOCAL_ADDRESS_IM"'/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_asset_usd_values"][0]["value"]["amount"]="9000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_asset_usd_values"][1]["key"]="minute/'"$LOCAL_ADDRESS_IM"'/0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["operator"]["operator_asset_usd_values"][1]["value"]["amount"]="3000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/delegation
	jq '.app_state["delegation"]["delegation_states"][0]["key"]="'"$LOCAL_ADDRESS_HEX"'_0x65/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65/'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][0]["states"]["undelegatable_share"]="4000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][0]["states"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][1]["key"]="'"$LOCAL_ADDRESS_HEX"'_0x65/0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65/'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][1]["states"]["undelegatable_share"]="1000000000000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][1]["states"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][2]["key"]="0xdc77c5b9d061ae2f7b35a6c8854b8f5bbea98eef_0x65/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65/'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][2]["states"]["undelegatable_share"]="5000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["delegation_states"][2]["states"]["pending_undelegation_amount"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["associations"][0]["staker_id"]="'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["associations"][0]["operator"]="'"$LOCAL_ADDRESS_IM"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["stakers_by_operator"][0]="'"$LOCAL_ADDRESS_IM"'/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65/'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["stakers_by_operator"][1]="'"$LOCAL_ADDRESS_IM"'/0xdac17f958d2ee523a2206206994597c13d831ec7_0x65/0xdc77c5b9d061ae2f7b35a6c8854b8f5bbea98eef_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["delegation"]["stakers_by_operator"][2]="'"$LOCAL_ADDRESS_IM"'/0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65/'"$LOCAL_ADDRESS_HEX"'_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/dogfood
	# for easy testing, use an epoch of 1 minute and 5 epochs until unbonded.
	# i did not use 1 epoch to allow for testing that it does not happen at each epoch.
	jq '.app_state["dogfood"]["params"]["asset_ids"][0]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["asset_ids"][1]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["epoch_identifier"]="minute"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["epochs_until_unbonded"]="5"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["val_set"][0]["public_key"]="'"$CONSENSUS_KEY"'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["val_set"][0]["power"]="12000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["last_total_power"]="12000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["dogfood"]["params"]["min_self_delegation"]="100"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/immint
	# set the epoch identifier to `minute`. the default set by the module is `day`,
	# which is more suitable for a live network and not a localnet.
	jq '.app_state["immint"]["params"]["epoch_identifier"]="minute"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/feedistribution
	# replace the avsAddr with the correct one
	jq --arg avs "$AVS_ADDRESS" '.app_state["feedistribution"]["all_avs_reward_assets"][0]["avs"] = $avs' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# x/oracle
	# chain
	jq '.app_state["oracle"]["params"]["chains"][1]["name"]="Ethereum"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["chains"][1]["desc"]="-"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["chains"][2]["name"]="Solana"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["chains"][2]["desc"]="-"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# token
	jq '.app_state["oracle"]["params"]["tokens"][1]["name"]="USDT"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["chain_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["contract_address"]="0xdac17f958d2ee523a2206206994597c13d831ec7"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["decimal"]="6"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["active"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][1]["asset_id"]="0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["oracle"]["params"]["tokens"][2]["name"]="NSTETH"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][2]["chain_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][2]["contract_address"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][2]["decimal"]="18"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][2]["active"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][2]["asset_id"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["oracle"]["params"]["tokens"][3]["name"]="NSTSOL"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][3]["chain_id"]="2"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][3]["contract_address"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][3]["decimal"]="9"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][3]["active"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["tokens"][3]["asset_id"]="0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x123"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# sources
	jq '.app_state["oracle"]["params"]["sources"][1]["name"]="Chainlink"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["sources"][1]["entry"]["offchain"]["0"]=""' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["sources"][1]["entry"]["onchain"]={}' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["sources"][1]["valid"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["sources"][1]["deterministic"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# rules
	jq '.app_state["oracle"]["params"]["rules"][2]["source_ids"][0]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["rules"][2]["nom"]=null' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["oracle"]["params"]["rules"][3]["source_ids"][0]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["rules"][3]["nom"]["source_ids"][0]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["rules"][3]["nom"]["minimum"]=1' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# token feeder
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["token_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["rule_id"]="2"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["start_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["start_base_block"]="2"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["interval"]="7"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][1]["end_block"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["oracle"]["params"]["token_feeders"][2]["token_id"]="2"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][2]["rule_id"]="3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][2]["start_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][2]["start_base_block"]="5"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][2]["interval"]="10"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][2]["end_block"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["piece_size_byte"]="32"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	jq '.app_state["oracle"]["params"]["token_feeders"][3]["token_id"]="3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][3]["rule_id"]="3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][3]["start_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][3]["start_base_block"]="5"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][3]["interval"]="7"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["token_feeders"][3]["end_block"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["piece_size_byte"]="32"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["max_size_prices"]="3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["params"]["epoch_identifier"]="minute"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# USDT
	jq '.app_state["oracle"]["prices_list"][0]["next_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][0]["price_list"][0]["decimal"]="6"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][0]["price_list"][0]["price"]="1000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][0]["price_list"][0]["round_id"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][0]["price_list"][0]["timestamp"]="2025-11-07 01:00:00"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][0]["token_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# NSTETH
	jq '.app_state["oracle"]["prices_list"][1]["next_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][1]["price_list"][0]["decimal"]="6"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][1]["price_list"][0]["price"]="3000000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][1]["price_list"][0]["round_id"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][1]["price_list"][0]["timestamp"]="2025-11-07 01:00:00"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][1]["token_id"]="2"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# NSTSOL
	jq '.app_state["oracle"]["prices_list"][2]["next_round_id"]="1"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][2]["price_list"][0]["decimal"]="6"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][2]["price_list"][0]["price"]="155000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][2]["price_list"][0]["round_id"]="0"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][2]["price_list"][0]["timestamp"]="2025-11-07 01:00:00"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["oracle"]["prices_list"][2]["token_id"]="3"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# custom epoch definitions can be added here, if required.
	# see https://github.com/imua-xyz/imuachain/blob/82b2509ad33ab7679592dcb1aa56a7a811128410/local_node.sh#L123 as an example

	# generate oracle_env_chainlink.yaml file
	oracle_env_chainlink_content=$(
		cat <<EOF
urls:
  mainnet: !!str https://eth-mainnet.g.alchemy.com/v2/${ALCHEMY_API_KEY}
  sepolia: !!str https://eth-sepolia.g.alchemy.com/v2/${ALCHEMY_API_KEY}
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
tokens:
  - token: ETHUSDT
    sources: chainlink
  - token: NSTETH
    sources: beaconchain
  - token: WSTETH
    sources: chainlink
  - token: NSTSOL
    sources: solana
sender:
  path: $HOMEDIR/config
imua:
  chainid: $CHAINID
  appName: imua
  grpc: 127.0.0.1:9090
  ws: !!str ws://127.0.0.1:26657/websocket
  rpc: !!str http://127.0.0.1:26657
status:
  grpc: 50052
#debugger:
#  grpc: !!str :50051
EOF
	)

	# Write the YAML content to a file
	echo "$oracle_feeder_content" >"$ORACLE_FEEDER"

	# generate oracle_env_beaconchain.yaml
	oracle_env_beaconchain_content=$(
		cat <<EOF
urls:
  beaconchain: !!str https://ethereum-holesky-beacon-api.publicnode.com
  eth: !!str https://eth-holesky.g.alchemy.com/v2/${ALCHEMY_API_KEY}
nstid:
  !!str 0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65
bootstrap: !!str ${BOOTSTRAP}
EOF
	)
	# Write the YAML content to a file
	echo "$oracle_env_beaconchain_content" >"$ORACLE_ENV_BEACONCHAIN"

	# generate oracle_env_solana.yaml
	oracle_env_solana_content=$(
		cat <<EOF
url:
  !!str https://solana-mainnet.g.alchemy.com/v2/${ALCHEMY_API_KEY}
nstid:
  !!str 0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x123
bootstrap: !!str 0x38674073a3713dd2C46892f1d2C5Dadc5Bb14172
EOF
	)
	# Write the YAML content to a file
	echo "$oracle_env_solana_content" >"$ORACLE_ENV_SOLANA"

	if [[ $1 == "pending" ]]; then
		$SED_CMD 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
		$SED_CMD 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
		$SED_CMD 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
		$SED_CMD 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
		$SED_CMD 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
		$SED_CMD 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
		$SED_CMD 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
		$SED_CMD 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
	fi

	# remove evmos seeds for localnet
	$SED_CMD 's/seeds = "[^"]*"/seeds = ""/' "$CONFIG"

	# enable prometheus metrics
	$SED_CMD 's/prometheus = false/prometheus = true/' "$CONFIG"
	$SED_CMD 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
	$SED_CMD 's/enabled = false/enabled = true/g' "$APP_TOML"
	$SED_CMD 's/enable = false/enable = true/g' "$APP_TOML"

	# Change proposal periods to pass within a reasonable time for local testing
	$SED_CMD 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$HOMEDIR"/config/genesis.json
	$SED_CMD 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$HOMEDIR"/config/genesis.json

	# set custom pruning settings for localnet
	$SED_CMD 's/pruning = "default"/pruning = "nothing"/g' "$APP_TOML"

	# make sure the localhost IP is 0.0.0.0
	$SED_CMD 's/127.0.0.1/0.0.0.0/g' "$CONFIG"
	$SED_CMD 's/localhost/0.0.0.0/g' "$CONFIG"
	$SED_CMD 's/localhost/0.0.0.0/g' "$APP_TOML"
	$SED_CMD 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

	# Allocate genesis accounts (cosmos formatted addresses)
	for KEY in "${KEYS[@]}"; do
		imuad add-genesis-account "$KEY" 100000000000000000000000000hua --keyring-backend "$KEYRING" --home "$HOMEDIR"
	done
	imuad add-genesis-account "${LOCAL_NAME}" 100000000000000000000000000hua --keyring-backend "$KEYRING" --home "$HOMEDIR"

	# bc is required to add these big numbers
	# note the extra +1 is for LOCAL_NAME
	total_supply=$(echo "(${#KEYS[@]} + 1) * 100000000000000000000000000" | bc)
	jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	imuad validate-genesis --home "$HOMEDIR"

	if [[ $1 == "pending" ]]; then
		echo "pending mode is on, please wait for the first block committed."
	fi
fi

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
# imuad start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001hua --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --json-rpc.enable true --home "$HOMEDIR" --chain-id "$CHAINID" --grpc.enable true --oracle
imuad start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001hua --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --json-rpc.enable true --home "$HOMEDIR" --chain-id "$CHAINID" --grpc.enable true --oracle --feeder_log_path "$HOMEDIR/logs"

# imuad start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001hua --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --json-rpc.enable true --home "$HOMEDIR" --chain-id "$CHAINID" --grpc.enable true --oracle --feeder_bin /Users/linqing/workplace/github.com/leonz/imua-xyz/price-feeder/build/price-feeder
# imuad start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001hua --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --json-rpc.enable true --home "$HOMEDIR" --chain-id "$CHAINID" --grpc.enable true --oracle --feeder_bin /Users/linqing/workplace/github.com/leonz/imua-xyz/price-feeder/build/price-feeder --feeder_log_path "$HOMEDIR/logs"
