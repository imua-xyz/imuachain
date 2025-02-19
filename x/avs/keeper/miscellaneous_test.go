package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

func TestReceiptMarshalBinary(t *testing.T) {
	task := types.TaskResponse{
		TaskID:    10,
		NumberSum: big.NewInt(1000),
	}

	packed, err := types.Args.Pack(&task)
	if err != nil {
		fmt.Println("bad bad ", err)
		return
	} else {
		fmt.Println("abi encoded", hexutil.Encode(packed))
	}

	args := make(map[string]interface{})

	err = types.Args.UnpackIntoMap(args, packed)
	result, _ := types.Args.Unpack(packed)
	fmt.Println("unpacked", result[0])
	hash := crypto.Keccak256Hash(packed)
	fmt.Println("hash:", hash.String())

	key := args["TaskResponse"]
	fmt.Println("key", key)
	for _, elem := range result {
		switch v := elem.(type) {
		case uint64:
			fmt.Println("Found uint64:", v)
		case *big.Int:
			fmt.Println("Found *big.Int:", v)
		case *types.TaskResponse:
			fmt.Println("types.TaskResponse type found")
		default:
			fmt.Println("Unknown type found")
		}
	}
	taskNew, _ := result[0].(*types.TaskResponse)
	fmt.Println("hash:", taskNew)

	var taskResponse types.TaskResponse

	if err := types.Args.Copy(&taskResponse, result); err != nil {
		fmt.Println("unpacked", result)
	}
	fmt.Println("taskResponse", taskResponse)
}

func Test_difference(t *testing.T) {
	arr1 := []string{"apple", "banana", "cherry"}
	arr2 := []string{"apple", "cherry", "date"}

	diff := types.Difference(arr1, arr2)
	fmt.Println("Differences:", diff)
	num1 := sdk.MustNewDecFromStr("1.3")
	num2 := sdk.MustNewDecFromStr("12.3")

	// Perform division
	result := num1.Quo(num2)

	// Convert result to percentage
	percentage := result.Mul(sdk.NewDec(100))

	// Print the result
	fmt.Print(percentage)
}

func Test_genKey(t *testing.T) {
	addresses := make([]string, 5)

	for i := 0; i < 5; i++ {
		address := utiltx.GenerateAddress()
		exoAddress := sdk.AccAddress(address.Bytes()).String()
		addresses[i] = exoAddress
	}

	fmt.Println("Generated EXO addresses:")
	for _, address := range addresses {
		fmt.Println(address)
	}
}
