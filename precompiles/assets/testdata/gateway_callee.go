package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed GatewayCallee.json
	GatewayCalleeJSON []byte

	// GatewayCalleeContract is the compiled contract called by the gateway
	GatewayCalleeContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(GatewayCalleeJSON, &GatewayCalleeContract)
	if err != nil {
		panic(err)
	}

	if len(GatewayCalleeContract.Bin) == 0 {
		panic("failed to load GatewayCallee")
	}
}
