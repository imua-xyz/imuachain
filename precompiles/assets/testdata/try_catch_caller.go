package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed TryCatchCaller.json
	TryCatchCallerJSON []byte

	// TryCatchCallerContract is the compiled contract calling the
	// PrecompileCallerThatReverts contract, which acts as gateway.
	TryCatchCallerContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(TryCatchCallerJSON, &TryCatchCallerContract)
	if err != nil {
		panic(err)
	}

	if len(TryCatchCallerContract.Bin) == 0 {
		panic("failed to load TryCatchCaller")
	}
}
