package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed PrecompileCallerThatReverts.json
	PrecompileCallerThatRevertsJSON []byte

	// PrecompileCallerThatRevertsContract is contract which calls the
	// precompile. It acts as gateway.
	PrecompileCallerThatRevertsContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(PrecompileCallerThatRevertsJSON, &PrecompileCallerThatRevertsContract)
	if err != nil {
		panic(err)
	}

	if len(PrecompileCallerThatRevertsContract.Bin) == 0 {
		panic("failed to load PrecompileCallerThatReverts")
	}
}
