package types

import (
	evmtype "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/utils"
)

// ImuachainAvailableEVMExtensions defines the default active precompiles
var (
	// DefaultEVMDenom defines the default EVM denomination on Imuachain
	DefaultEVMDenom                 = utils.BaseDenom
	ImuachainAvailableEVMExtensions = []string{
		"0x0000000000000000000000000000000000000400", // bech32 precompile
		// "0x0000000000000000000000000000000000000802", // ICS20 transfer precompile
		"0x0000000000000000000000000000000000000804", // assets precompile
		"0x0000000000000000000000000000000000000805", // delegation precompile
		"0x0000000000000000000000000000000000000806", // reward precompile
		// "0x0000000000000000000000000000000000000807", // slash precompile
		"0x0000000000000000000000000000000000000809", // bls precompile
		"0x0000000000000000000000000000000000000901", // avs precompile
	}
)

// ImuachainEvmDefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func ImuachainEvmDefaultParams() evmtype.Params {
	return evmtype.Params{
		EvmDenom:            DefaultEVMDenom,
		EnableCreate:        evmtype.DefaultEnableCreate,
		EnableCall:          evmtype.DefaultEnableCall,
		ChainConfig:         evmtype.DefaultChainConfig(),
		ExtraEIPs:           nil,
		AllowUnprotectedTxs: evmtype.DefaultAllowUnprotectedTxs,
		ActivePrecompiles:   ImuachainAvailableEVMExtensions,
	}
}
