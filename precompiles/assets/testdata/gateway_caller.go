package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed GatewayCaller.json
	GatewayCallerJSON []byte

	// GatewayCallerContract is the compiled contract calling the gateway
	GatewayCallerContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(GatewayCallerJSON, &GatewayCallerContract)
	if err != nil {
		panic(err)
	}

	if len(GatewayCallerContract.Bin) == 0 {
		panic("failed to load GatewayCaller")
	}
}
