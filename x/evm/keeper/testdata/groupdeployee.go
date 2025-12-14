package testdata

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

type CompiledContract struct {
	ABI        abi.ABI            `json:"abi"`
	Bin        evmtypes.HexString `json:"bin"`
	BinRuntime evmtypes.HexString `json:"bin-runtime"`
}

type jsonCompiledContract struct {
	ABI        string             `json:"abi"`
	Bin        evmtypes.HexString `json:"bin"`
	BinRuntime evmtypes.HexString `json:"bin-runtime"`
}

var (
	//go:embed GroupDeployee.json
	GroupDeployeeJSON []byte

	// GroupDeployeeContract is the compiled contract
	GroupDeployeeContract CompiledContract
)

func init() {
	err := json.Unmarshal(GroupDeployeeJSON, &GroupDeployeeContract)
	if err != nil {
		panic(err)
	}

	if len(GroupDeployeeContract.Bin) == 0 {
		panic("failed to load GroupDeployee.Bin")
	}

	if len(GroupDeployeeContract.BinRuntime) == 0 {
		panic("failed to load GroupDeployee.BinRuntime")
	}
}

// MarshalJSON serializes ByteArray to hex
func (s CompiledContract) MarshalJSON() ([]byte, error) {
	abi1, err := json.Marshal(s.ABI)
	if err != nil {
		return nil, err
	}
	return json.Marshal(
		jsonCompiledContract{
			ABI:        string(abi1),
			Bin:        s.Bin,
			BinRuntime: s.BinRuntime,
		},
	)
}

// UnmarshalJSON deserializes ByteArray to hex
func (s *CompiledContract) UnmarshalJSON(data []byte) error {
	var x jsonCompiledContract
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}

	s.Bin = x.Bin
	s.BinRuntime = x.BinRuntime
	if err := json.Unmarshal([]byte(x.ABI), &s.ABI); err != nil {
		return fmt.Errorf("failed to unmarshal ABI: %w", err)
	}

	return nil
}
