package testdata

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

var (
	//go:embed ThirdPartyCallee.json
	ThirdPartyCalleeJSON []byte

	// ThirdPartyCalleeContract receives funds; only used to revert by
	// sending more funds than the caller has.
	ThirdPartyCalleeContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(ThirdPartyCalleeJSON, &ThirdPartyCalleeContract)
	if err != nil {
		panic(err)
	}

	if len(ThirdPartyCalleeContract.Bin) == 0 {
		panic("failed to load ThirdPartyCallee")
	}
}
