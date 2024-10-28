#!/usr/bin/env bash

set -e
set -x

echo "Building subscriberd..."
go install ./cmd/subscriberd

echo "Registering operators..."
for i in {2..4}; do
    TX_HASH=$(yes | exocored tx operator register-operator \
        --meta-info operator$i \
        --commission-rate 0 \
        --commission-max-rate 1 \
        --commission-max-change-rate 1 \
        --from dev$((i-2)) \
        --home ~/.tmp-exocored/ \
        --gas-prices 700000000hua \
        --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
    while true; do
        TX_STATUS=$(exocored query tx $TX_HASH --output json | jq -r .code)
        if [ "$TX_STATUS" == "0" ]; then
            echo "Transaction $TX_HASH confirmed in a block!"
            break
        else
            echo "Waiting for transaction $TX_HASH to be included in a block..."
        fi
        sleep 1
    done
done

echo "Making deposits, delegations and associations..."
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

echo "Initializing subscriberd folders..."
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

echo "Registering the subscriber chain..."
TX_HASH=$(yes | exocored tx coordinator register-subscriber-chain \
    --from dev0 \
    --home ~/.tmp-exocored \
    --gas-prices 1000000000hua \
    "$(cat app/subscriber/dumpling-1.json | jq -c)" \
    --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
while true; do
    TX_STATUS=$(exocored query tx $TX_HASH --output json | jq -r .code)
    if [ "$TX_STATUS" == "0" ]; then
        echo "Transaction $TX_HASH confirmed in a block!"
        break
    else
        echo "Waiting for transaction $TX_HASH to be included in a block..."
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
        --home ~/.tmp-exocored \
        --gas-prices 700000000hua \
    --output json | jq -r .txhash | tr -d '\n' | cut -c 5-)
    while true; do
        TX_STATUS=$(exocored query tx $TX_HASH --output json | jq -r .code)
        if [ "$TX_STATUS" == "0" ]; then
            echo "Transaction $TX_HASH confirmed in a block!"
            break
        else
            echo "Waiting for transaction $TX_HASH to be included in a block..."
        fi
        sleep 1
    done
done