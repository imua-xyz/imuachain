# Subscriber Chain Example

This is an example of the subscriber chain which may be used for testing. A subscriber chain is a chain that is an AVS
on Exocore. As an AVS, the chain receives security from PoS operators registered on Exocore. The chain must incentivize
operators to validate on it, by offering rewards which accrue to these operators and stakers delegating to them.

## Testing steps

1. Start a network using `./local_node.sh`
2. Install `subscriberd` binary by `go install ./cmd/subscriberd`
3. Register 3 operators

```shell
for i in {2..4}; do
    yes | exocored tx operator register-operator \
        --meta-info operator$i \
        --commission-rate 0 \
        --commission-max-rate 1 \
        --commission-max-change-rate 1 \
        --from dev$((i-2)) \
        --home ~/.tmp-exocored/ \
        --gas-prices 700000000hua
done
```

4. Make self-delegations to these operators using `cast`. It is a 3-step process: make (fake) deposit, then delegation
and then the association with the operator. Our codebase intentionally does not assume any associations by itself.

```shell
ETH_PRIVATE_KEY=$(exocored keys unsafe-export-eth-key local_funded_account --home ~/.tmp-exocored --keyring-backend test)
LOCAL_ACCOUNT=$(cast wallet a $ETH_PRIVATE_KEY)
ASSETS_PRECOMPILE=0x0000000000000000000000000000000000000804
DELEGATION_PRECOMPILE=0x0000000000000000000000000000000000000805
ETH_LZ_ID=101
EXO_ETH_RPC_URL=http://localhost:8545
QUANTITY=100
TOKEN_ADDRESS=0xdAC17F958D2ee523a2206206994597C13D831ec7
NONCE=$(cast nonce --rpc-url $EXO_ETH_RPC_URL $LOCAL_ACCOUNT)
for key in dev0 dev1 dev2; do
    EXO_ADDRESS=$(exocored keys show -a $key --home ~/.tmp-exocored)
    ETH_ADDRESS=$(exocored keys parse $EXO_ADDRESS --output json | jq -r .bytes | cast 2a)
    # deposit
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $ASSETS_PRECOMPILE \
        "depositLST(uint32,bytes,bytes,uint256)" \
        $ETH_LZ_ID \
        $(cast 2b $TOKEN_ADDRESS) \
        $ETH_ADDRESS \
        $(cast 2w $QUANTITY) \
        --private-key $ETH_PRIVATE_KEY \
        --async \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
    # delegate
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $DELEGATION_PRECOMPILE \
        "delegate(uint32,uint64,bytes,bytes,bytes,uint256)" \
        $ETH_LZ_ID \
        $NONCE \
        $(cast 2b $TOKEN_ADDRESS) \
        $(cast 2b $ETH_ADDRESS) \
        $(cast fu $EXO_ADDRESS) \
        $(cast 2w $QUANTITY) \
        --private-key $ETH_PRIVATE_KEY \
        --async \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
    # associate
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $DELEGATION_PRECOMPILE \
        "associateOperatorWithStaker(uint32,bytes,bytes)" \
        $ETH_LZ_ID \
        $(cast 2b $ETH_ADDRESS) \
        $(cast fu $EXO_ADDRESS) \
        --private-key $ETH_PRIVATE_KEY \
        --async \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
done
```

5. Initialize a `subscriberd` folder thrice (for the 3 operators).

```shell
CHAIN_ID="dumpling-1"
KEYRING="test"
MONIKER="localtestnet"
KEYS[0]="subscriber_operator1"
KEYS[1]="subscriber_operator2"
KEYS[2]="subscriber_operator3"

for KEY in "${KEYS[@]}"; do
    HOMEDIR="$HOME/.tmp-subscriberd/$KEY"
    subscriberd init $MONIKER -o --chain-id $CHAIN_ID --home "$HOMEDIR" --default-denom subcoin
done
```

6. Create the subscriber chain params as a json: `dumpling-1.json`. The durations are measured in seconds.

```json
{
  "chain_id": "dumpling-1",
  "epoch_identifier": "minute",
  "asset_ids": [
    "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65",
    "0x0000000000000000000000000000000000000000_0x0"
  ],
  "max_validators": 50,
  "min_self_delegation_usd": "100",
  "subscriber_params": {
    "coordinator_fee_pool_addr_str": "",
    "distribution_transmission_channel": "",
    "blocks_per_distribution_transmission": "40",
    "subscriber_redistribution_fraction": "0.85",
    "reward_denom": "subcoin",
    "ibc_timeout_period": "2419200",
    "transfer_timeout_period": "3600",
    "unbonding_period": "1728000",
    "historical_entries": "1000",
    "slash_fraction_downtime": "0.01",
    "downtime_jail_duration": "600",
    "slash_fraction_double_sign": "0.05"
  }
}
```

7. Register the subscriber chain.

```shell
yes | exocored tx coordinator register-subscriber-chain \
    --from dev0 \
    --home ~/.tmp-exocored \
    --gas-prices 1000000000hua \
    "$(cat app/subscriber/dumpling-1.json | jq -c)"
```

8. Opt in to the subscriber chain for all 3 operators.

```shell
# Redeclared the following items for safety
CHAIN_ID="dumpling-1"
KEYS[0]="subscriber_operator1"
KEYS[1]="subscriber_operator2"
KEYS[2]="subscriber_operator3"
# Actual script
AVS_ADDRESS=$(exocored query avs AVSAddrByChainID $CHAIN_ID --node $EXOCORE_COS_GRPC_URL --output json | jq -r .avs_address)
for i in {0..2}; do
    key="dev$i"
    HOMEDIR="$HOME/.tmp-subscriberd/${KEYS[$i]}"
    yes | exocored tx operator opt-into-avs \
        $AVS_ADDRESS \
        $(subscriberd --home $HOMEDIR tendermint show-validator) \
        --from $key \
        --home ~/.tmp-exocored \
        --gas-prices 700000000hua
done
```

9. Wait for the next `minute` epoch to begin and pull the generated subscriber module genesis state from the coordinator.
