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
	//go:embed Deployed.json
	DeployedJSON []byte

	// DeployedContract is the compiled contract
	DeployedContract CompiledContract
)

func init() {
	err := json.Unmarshal(DeployedJSON, &DeployedContract)
	if err != nil {
		panic(err)
	}

	if len(DeployedContract.Bin) == 0 {
		panic("failed to load Deployed.Bin")
	}

	if len(DeployedContract.BinRuntime) == 0 {
		panic("failed to load Deployed.BinRuntime")
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
