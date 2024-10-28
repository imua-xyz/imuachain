package keeper_test

import (
	"math/big"
	"reflect"
	"testing"

	utiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestReceiptMarshalBinary(t *testing.T) {
	task := types.TaskResponse{
		TaskID:    10,
		NumberSum: big.NewInt(1000),
	}

	packed, err := types.Args.Pack(&task)
	if err != nil {
		t.Errorf("Error packing task: %v", err)
		return
	} else {
		t.Logf("ABI encoded: %s", hexutil.Encode(packed))
	}

	args := make(map[string]interface{})

	err = types.Args.UnpackIntoMap(args, packed)
	result, _ := types.Args.Unpack(packed)
	t.Logf("Unpacked: %v", result[0])
	hash := crypto.Keccak256Hash(packed)
	t.Logf("Hash: %s", hash.String())

	key := args["TaskResponse"]
	t.Logf("Key: %v", key)
}

func Test_difference(t *testing.T) {
	arr1 := []string{"apple", "banana", "cherry"}
	arr2 := []string{"apple", "cherry", "date"}

	diff := types.Difference(arr1, arr2)
	expectedDiff := []string{"banana", "date"}
	if !reflect.DeepEqual(diff, expectedDiff) {
		t.Errorf("Expected difference %v, got %v", expectedDiff, diff)
	} else {
		t.Logf("Differences: %v", diff)
	}

	num1 := sdk.MustNewDecFromStr("1.3")
	num2 := sdk.MustNewDecFromStr("12.3")

	// Perform division
	result := num1.Quo(num2)

	// Convert result to percentage
	percentage := result.Mul(sdk.NewDec(100))

	expectedPercentage := sdk.MustNewDecFromStr("10.569105691056910600")
	if !percentage.Equal(expectedPercentage) {
		t.Errorf("Expected percentage %s, got %s", expectedPercentage, percentage)
	} else {
		t.Logf("Percentage: %s", percentage)
	}
}

func Test_genKey(t *testing.T) {
	addresses := make([]string, 5)

	for i := 0; i < 5; i++ {
		address := utiltx.GenerateAddress()
		exoAddress := sdk.AccAddress(address.Bytes()).String()
		addresses[i] = exoAddress
	}

	t.Log("Generated EXO addresses:")
	for _, address := range addresses {
		t.Log(address)
		if _, err := sdk.AccAddressFromBech32(address); err != nil {
			t.Errorf("Invalid EXO address: %s", address)
		}
	}
}
