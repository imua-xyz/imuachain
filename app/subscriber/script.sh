#!/usr/bin/env bash

HOMEROOT="$HOME/.tmp-subscriberd"
EXO_HOMEDIR="$HOME/.tmp-exocored"
SUBSCRIBER_JSON="app/subscriber/dumpling-1.json"

set -e

echo "Building subscriberd..."
go install ./cmd/subscriberd

echo "Waiting for network to start..."
while true; do
    price=$(exocored query oracle show-prices 1 --output json 2>/dev/null | jq -r '.prices.price_list.[-1].price')
    if [[ ! "$price" =~ ^[0-9]+$ ]]; then
        echo "Failed to retrieve USDT price. Retrying..."
    elif [ "$price" -ge 99900000 ]; then
        echo "USDT price available: $price"
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
ETH_PRIVATE_KEY=$(exocored keys unsafe-export-eth-key local_funded_account --home $EXO_HOMEDIR --keyring-backend test)
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
KEYRING="test"
MONIKER="localtestnet"
KEYS[0]="subscriber_operator1"
KEYS[1]="subscriber_operator2"
KEYS[2]="subscriber_operator3"
rm -rf $HOME/.tmp-subscriberd
mkdir -p $HOME/.tmp-subscriberd
for KEY in "${KEYS[@]}"; do
    HOMEDIR="$HOME/.tmp-subscriberd/$KEY"
    subscriberd init $MONIKER -o --chain-id $CHAIN_ID --home "$HOMEDIR" --default-denom subcoin
done

# The subscriber chain must be registered immediately after the epoch increases
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
for i in {0..2}; do
    key="dev$i"
    HOMEDIR="$HOME/.tmp-subscriberd/${KEYS[$i]}"
    TX_HASH=$(yes | exocored tx operator opt-into-avs \
        $AVS_ADDRESS \
        $(subscriberd --home $HOMEDIR tendermint show-validator) \
        --from $key \
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