package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed UnknownMethodCaller.json
	UnknownMethodCallerJSON []byte

	// UnknownMethodCallerContract is the compiled contract calling the precompiles
	UnknownMethodCallerContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(UnknownMethodCallerJSON, &UnknownMethodCallerContract)
	if err != nil {
		panic(err)
	}

	if len(UnknownMethodCallerContract.Bin) == 0 {
		panic("failed to load UnknownMethodCaller")
	}
}
