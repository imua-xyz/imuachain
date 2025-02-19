package types_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/imua-xyz/imuachain/x/evm/types"
)

const (
	url = "https://rpc.ankr.com/eth_sepolia"
)

// TestValidateDefaultPredeploys checks the format of the default predeploys, and that
// the code at the provided predeploys matches the code at the remote URL.
func TestValidateDefaultPredeploys(t *testing.T) {
	// do not use the helper functions like GetByteAddress and GetByteCode intentionally.
	// some of this code may overlap init() but duplicating it with the slight differences
	// offers stronger guarantees of the validity (as test and during runtime).
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
		// remote code starts with 0x, so use hexutil.
		// also contains a blank string check.
		parsedRemoteCode, err := hexutil.Decode(remoteCode)
		if err != nil {
			t.Fatalf("error parsing remote code for address %s: %v", address, err)
		}
		// local code does not start with 0x, so use common.Hex2Bytes
		// this is because we follow the convention in x/evm/genesis.go
		// since remoteCode is not blank, no need to check if localCode is blank.
		parsedLocalCode := common.Hex2Bytes(code)
		// for different lengths, use bytes.Equal. otherwise string is faster.
		if !bytes.Equal(parsedLocalCode, parsedRemoteCode) {
			t.Fatalf("predeploy code for address %s does not match remote code", address)
		}
	}
}

// getCode calls eth_getCode at the given address for the latest block.
// it uses the fixed global URL defined in this file.
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
