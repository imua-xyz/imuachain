package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed Gateway.json
	GatewayJSON []byte

	// GatewayContract is the compiled contract calling the assets precompile
	GatewayContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(GatewayJSON, &GatewayContract)
	if err != nil {
		panic(err)
	}

	if len(GatewayContract.Bin) == 0 {
		panic("failed to load Gateway")
	}
}
