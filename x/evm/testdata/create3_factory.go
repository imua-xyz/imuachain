package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed ICreate3Factory.json
	Create3FactoryJSON []byte

	// Create3FactoryContract is the compiled contract
	Create3FactoryContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(Create3FactoryJSON, &Create3FactoryContract)
	if err != nil {
		panic(err)
	}
	// it is an interface so there is no bin
}
