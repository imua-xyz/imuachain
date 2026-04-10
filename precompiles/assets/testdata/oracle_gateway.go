package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed OracleGateway.json
	oracleGatewayJSON []byte

	// OracleGatewayContract is the compiled contract for oracle-bridge tests.
	OracleGatewayContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(oracleGatewayJSON, &OracleGatewayContract)
	if err != nil {
		panic(err)
	}

	if len(OracleGatewayContract.Bin) == 0 {
		panic("failed to load OracleGateway")
	}
}
