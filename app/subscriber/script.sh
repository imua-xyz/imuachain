#!/usr/bin/env bash

HOMEROOT="$HOME/.tmp-subscriberd"
EXO_HOMEDIR="$HOME/.tmp-exocored"
SUBSCRIBER_JSON="app/subscriber/dumpling-1.json"
KEYRING_BACKEND="test"

set -e

echo "Building subscriberd..."
go install ./cmd/subscriberd

echo "Waiting for network to start..."
while true; do
    PRICE=$(exocored query oracle show-prices 1 --output json 2>/dev/null | jq -r '.prices.price_list.[-1].price')
    if [[ ! "$PRICE" =~ ^[0-9]+$ ]]; then
        echo "Failed to retrieve USDT price. Retrying..."
    elif [ "$PRICE" -ge 99900000 ]; then
        echo "USDT price available: $PRICE"
        break
    else
        echo "Waiting for network to start..."
    fi
    sleep 1
done

echo "Registering operators..."
for i in {2..4}; do
    TX_HASH=$(yes | exocored tx operator register-operator \
        --meta-info operator$i \
        --commission-rate 0 \
        --commission-max-rate 1 \
        --commission-max-change-rate 1 \
        --from dev$((i-2)) \
        --home $EXO_HOMEDIR \
        --gas-prices 700000000hua \
        --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
    while true; do
        TX_STATUS=$(exocored query tx $TX_HASH --output json 2>/dev/null | jq -r .code)
        if [[ ! "$TX_STATUS" =~ ^[0-9]+$ ]]; then
            echo "Waiting for transaction $TX_HASH to be included in a block..."
        elif [ "$TX_STATUS" == "0" ]; then
            echo "Transaction $TX_HASH confirmed in a block!"
            break
        else
            echo "Transaction $TX_HASH failed with code $TX_STATUS!"
            echo "Could not register operator$i"
            exit 1
        fi
        sleep 1
    done
done

echo "Making deposits, delegations and associations..."
ETH_PRIVATE_KEY=$(exocored keys unsafe-export-eth-key local_funded_account --home $EXO_HOMEDIR --keyring-backend $KEYRING_BACKEND)
LOCAL_ACCOUNT=$(cast wallet a $ETH_PRIVATE_KEY)
ASSETS_PRECOMPILE=0x0000000000000000000000000000000000000804
DELEGATION_PRECOMPILE=0x0000000000000000000000000000000000000805
ETH_LZ_ID=101
EXO_ETH_RPC_URL=http://localhost:8545
QUANTITY=100
TOKEN_ADDRESS=0xdAC17F958D2ee523a2206206994597C13D831ec7
NONCE=$(cast nonce --rpc-url $EXO_ETH_RPC_URL $LOCAL_ACCOUNT)
for key in dev0 dev1 dev2; do
    EXO_ADDRESS=$(exocored keys show -a $key --home $EXO_HOMEDIR)
    ETH_ADDRESS=$(exocored keys parse $EXO_ADDRESS --output json | jq -r .bytes | cast 2a)
    # deposit
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $ASSETS_PRECOMPILE \
        "depositLST(uint32,bytes,bytes,uint256) returns (bool, uint256)" \
        $ETH_LZ_ID \
        $(cast 2b $TOKEN_ADDRESS) \
        $ETH_ADDRESS \
        $(cast 2w $QUANTITY) \
        --private-key $ETH_PRIVATE_KEY \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
    # delegate
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $DELEGATION_PRECOMPILE \
        "delegate(uint32,uint64,bytes,bytes,bytes,uint256) returns (bool)" \
        $ETH_LZ_ID \
        $NONCE \
        $(cast 2b $TOKEN_ADDRESS) \
        $(cast 2b $ETH_ADDRESS) \
        $(cast fu $EXO_ADDRESS) \
        $(cast 2w $QUANTITY) \
        --private-key $ETH_PRIVATE_KEY \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
    # associate
    cast send --rpc-url $EXO_ETH_RPC_URL \
        $DELEGATION_PRECOMPILE \
        "associateOperatorWithStaker(uint32,bytes,bytes) returns (bool)" \
        $ETH_LZ_ID \
        $(cast 2b $ETH_ADDRESS) \
        $(cast fu $EXO_ADDRESS) \
        --private-key $ETH_PRIVATE_KEY \
        --nonce $NONCE
    NONCE=$((NONCE + 1))
done

echo "Initializing subscriberd folders..."
SUBSCRIBER_JSON="app/subscriber/dumpling-1.json"
CHAIN_ID=$(cat $SUBSCRIBER_JSON | jq -r .chain_id)
DENOM=$(cat $SUBSCRIBER_JSON | jq -r .subscriber_params.reward_denom)
./app/subscriber/init-nodes.py --binary $(which subscriberd) \
    --chain-id $CHAIN_ID \
    --log-level info \
    --folder $HOMEROOT/subscriber_operator \
    --mnemonics-file ./app/subscriber/mnemonics.txt \
    --denom $DENOM \
    --port-offset 5

# The subscriber chain must be registered immediately after the epoch increases so that we have enough time
# to opt into the chain before the next-to-next epoch starts. At that point, the subscriber chain genesis begins, so
# these operators must be available prior.
EPOCH_ID=$(cat $SUBSCRIBER_JSON | jq -r .epoch_identifier)
START_EPOCH_NUMBER=$(exocored query epochs current-epoch $EPOCH_ID --output json | jq -r .current_epoch)
while true; do
    CURRENT_EPOCH_NUMBER=$(exocored query epochs current-epoch $EPOCH_ID --output json 2>/dev/null | jq -r .current_epoch)
    if [[ ! "$CURRENT_EPOCH_NUMBER" =~ ^[0-9]+$ ]]; then
        echo "Failed to retrieve current epoch number. Retrying..."
    elif [ "$CURRENT_EPOCH_NUMBER" -gt "$START_EPOCH_NUMBER" ]; then
        echo "Current epoch number: $CURRENT_EPOCH_NUMBER (was $START_EPOCH_NUMBER)"
        break
    else
        echo "Waiting for epoch $EPOCH_ID to increase..."
    fi
    sleep 1
done

echo "Registering the subscriber chain..."
TX_HASH=$(yes | exocored tx coordinator register-subscriber-chain \
    --from dev0 \
    --home $EXO_HOMEDIR \
    --gas-prices 1000000000hua \
    "$(cat $SUBSCRIBER_JSON | jq -c)" \
    --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
while true; do
    TX_STATUS=$(exocored query tx $TX_HASH --output json 2>/dev/null | jq -r .code)
    if [[ ! "$TX_STATUS" =~ ^[0-9]+$ ]]; then
        echo "Waiting for transaction $TX_HASH to be included in a block..."
    elif [ "$TX_STATUS" == "0" ]; then
        echo "Transaction $TX_HASH confirmed in a block!"
        break
    else
        echo "Transaction $TX_HASH failed with code $TX_STATUS!"
        echo "Could not register the subscriber chain"
        exit 1
    fi
    sleep 1
done

echo "Opting in to the subscriber chain..."
AVS_ADDRESS=$(exocored query avs AVSAddrByChainID $CHAIN_ID --output json | jq -r .avs_address)
for i in {1..3}; do
    KEY="dev$((i - 1))"
    FOLDER="subscriber_operator$i"
    HOMEDIR="$HOMEROOT/$FOLDER"
    TX_HASH=$(yes | exocored tx operator opt-into-avs \
        $AVS_ADDRESS \
        $(subscriberd --home $HOMEDIR tendermint show-validator) \
        --from $KEY \
        --home $EXO_HOMEDIR \
        --gas-prices 700000000hua \
    --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
    while true; do
        TX_STATUS=$(exocored query tx $TX_HASH --output json 2>/dev/null | jq -r .code)
        if [[ ! "$TX_STATUS" =~ ^[0-9]+$ ]]; then
            echo "Waiting for transaction $TX_HASH to be included in a block..."
        elif [ "$TX_STATUS" == "0" ]; then
            echo "Transaction $TX_HASH confirmed in a block!"
            break
        else
            echo "Transaction $TX_HASH failed with code $TX_STATUS!"
            echo "Could not opt into the subscriber chain for $key"
            exit 1
        fi
        sleep 1
    done
done

# At the next epoch, the subscriber genesis state will be available
EPOCH_ID=$(cat $SUBSCRIBER_JSON | jq -r .epoch_identifier)
START_EPOCH_NUMBER=$(exocored query epochs current-epoch $EPOCH_ID --output json | jq -r .current_epoch)
while true; do
    CURRENT_EPOCH_NUMBER=$(exocored query epochs current-epoch $EPOCH_ID --output json 2>/dev/null | jq -r .current_epoch)
    if [[ ! "$CURRENT_EPOCH_NUMBER" =~ ^[0-9]+$ ]]; then
        echo "Failed to retrieve current epoch number. Retrying..."
    elif [ "$CURRENT_EPOCH_NUMBER" -gt "$START_EPOCH_NUMBER" ]; then
        echo "Current epoch number: $CURRENT_EPOCH_NUMBER (was $START_EPOCH_NUMBER)"
        break
    else
        echo "Waiting for epoch $EPOCH_ID to increase..."
    fi
    sleep 1
done

# Finally, make the genesis state
GENESIS=$(exocored query coordinator subscriber-genesis $CHAIN_ID --output json | jq .subscriber_genesis)
# these must be included manually otherwise they are dropped by the `jq --argjson` below
GENESIS=$(echo "$GENESIS" | jq '. + {coordinator_client_id: "", coordinator_channel_id: ""}')
GENESIS_FILE="$HOMEROOT/subscriber_operator1/config/genesis.json"
jq --argjson gen "$GENESIS" '.app_state.subscriber = $gen' "$GENESIS_FILE" > tmp.json && mv tmp.json "$GENESIS_FILE"
for i in {2..3}; do
    # copy everything over, including the genesis time
    cp "$GENESIS_FILE" "$HOMEROOT/subscriber_operator$i/config/genesis.json"
done
subscriberd start --home $HOMEROOT/subscriber_operator1 &
subscriberd start --home $HOMEROOT/subscriber_operator2 &
subscriberd start --home $HOMEROOT/subscriber_operator3 &