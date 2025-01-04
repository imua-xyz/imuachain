package types_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ExocoreNetwork/exocore/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	url = "https://rpc.ankr.com/eth_sepolia"
)

func TestValidateDefaultPredeploys(t *testing.T) {
	for _, predeploy := range types.DefaultPredeploys {
		address := predeploy.Address
		if !common.IsHexAddress(address) {
			t.Fatalf("predeploy address %s is not a valid hex address", address)
		}
		code := predeploy.Code
		if strings.HasPrefix(code, "0x") {
			t.Fatalf("predeploy code for address %s must not start with 0x", address)
		}
		remoteCode, err := getCode(address)
		if err != nil {
			t.Fatalf("error getting code for address %s: %v", address, err)
		}
		// remote code starts with 0x, so use hexutil
		parsedRemoteCode, err := hexutil.Decode(remoteCode)
		if err != nil {
			t.Fatalf("error parsing remote code for address %s: %v", address, err)
		}
		parsedLocalCode := common.Hex2Bytes(code)
		// for different lengths, use bytes.Equal. otherwise string is faster.
		if !bytes.Equal(parsedLocalCode, parsedRemoteCode) {
			t.Fatalf("predeploy code for address %s does not match remote code", address)
		}
	}
}

func getCode(address string) (string, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return "", err
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "eth_getCode", address, "latest")
	if err != nil {
		return "", err
	}

	return result, nil
}
