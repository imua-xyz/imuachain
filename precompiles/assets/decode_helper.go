package assets

import (
	"fmt"
	"math/big"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

// TypedArgs provides helper methods for safely asserting common argument types
type TypedArgs struct {
	args []interface{}
}

// NewTypedArgs creates a new TypedArgs instance
func NewTypedArgs(args []interface{}) *TypedArgs {
	return &TypedArgs{args: args}
}

func (ta *TypedArgs) RequireLen(expected int) error {
	if len(ta.args) != expected {
		return fmt.Errorf(cmn.ErrInvalidNumberOfArgs, expected, len(ta.args))
	}
	return nil
}

func (ta *TypedArgs) GetUint8(index int) (uint8, error) {
	if index >= len(ta.args) {
		return 0, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].(uint8)
	if !ok {
		return 0, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "uint8", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetUint32(index int) (uint32, error) {
	if index >= len(ta.args) {
		return 0, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].(uint32)
	if !ok {
		return 0, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "uint32", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetBytes(index int) ([]byte, error) {
	if index >= len(ta.args) {
		return nil, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].([]byte)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "[]byte", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetString(index int) (string, error) {
	if index >= len(ta.args) {
		return "", fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].(string)
	if !ok {
		return "", fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "string", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetBigInt(index int) (*big.Int, error) {
	if index >= len(ta.args) {
		return nil, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].(*big.Int)
	if !ok || val == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "*big.Int", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetEVMAddress(index int) (common.Address, error) {
	if index >= len(ta.args) {
		return common.Address{}, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "address", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetEVMAddressSlice(index int) ([]common.Address, error) {
	if index >= len(ta.args) {
		return nil, fmt.Errorf(exocmn.ErrIndexOutOfRange, index, len(ta.args))
	}
	val, ok := ta.args[index].([]common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "[]common.Address", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetPositiveUint8(index int) (uint8, error) {
	val, err := ta.GetUint8(index)
	if err != nil {
		return 0, err
	}
	if val == 0 {
		return 0, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "uint8", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetPositiveUint32(index int) (uint32, error) {
	val, err := ta.GetUint32(index)
	if err != nil {
		return 0, err
	}
	if val == 0 {
		return 0, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "uint32", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetRequiredBytes(index int) ([]byte, error) {
	val, err := ta.GetBytes(index)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "[]byte", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetRequiredString(index int) (string, error) {
	val, err := ta.GetString(index)
	if err != nil {
		return "", err
	}
	if len(val) == 0 {
		return "", fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "string", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetPositiveBigInt(index int) (*big.Int, error) {
	val, err := ta.GetBigInt(index)
	if err != nil {
		return nil, err
	}
	if val.Sign() <= 0 {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, index, "*big.Int", ta.args[index])
	}
	return val, nil
}

func (ta *TypedArgs) GetRequiredBytesPrefix(index int, length uint32) ([]byte, error) {
	val, err := ta.GetRequiredBytes(index)
	if err != nil {
		return nil, err
	}
	if len(val) < int(length) {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(val), length)
	}
	return val[:length], nil
}

func (ta *TypedArgs) GetRequiredEVMAddressSlice(index int) ([]common.Address, error) {
	val, err := ta.GetEVMAddressSlice(index)
	if err != nil {
		return nil, err
	}
	if len(val) == 0 {
		return nil, fmt.Errorf(exocmn.ErrEmptyGateways)
	}
	return val, nil
}

func (ta *TypedArgs) GetRequiredHexAddress(index int, addressLength uint32) (string, error) {
	val, err := ta.GetRequiredBytesPrefix(index, addressLength)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(val), nil
}
