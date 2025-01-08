package delegation

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

func (p Precompile) GetDelegationParamsFromInputs(ctx sdk.Context, args []interface{}) (*delegationtypes.DelegationOrUndelegationParams, error) {
	inputsLen := len(p.ABI.Methods[MethodDelegate].Inputs)
	if len(args) != inputsLen {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}

	delegationParams := &delegationtypes.DelegationOrUndelegationParams{}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "uint32", args[0])
	}
	delegationParams.ClientChainID = uint64(clientChainID)

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, delegationParams.ClientChainID)
	if err != nil {
		return nil, err
	}
	clientChainAddrLength := info.AddressLength

	// TODO: Remove this once the layerzero nonce is removed as an input parameter. This can be addressed
	// when we adjust the precompile interface.
	_, ok = args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "uint64", args[1])
	}

	// the length of client chain address inputted by caller is 32, so we need to check the length and remove the padding according to the actual length.
	assetAddr, ok := args[2].([]byte)
	if !ok || assetAddr == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 2, "[]byte", args[2])
	}
	// #nosec G115
	if uint32(len(assetAddr)) < clientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(assetAddr), clientChainAddrLength)
	}
	delegationParams.AssetsAddress = assetAddr[:clientChainAddrLength]

	stakerAddr, ok := args[3].([]byte)
	if !ok || stakerAddr == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 3, "[]byte", args[3])
	}
	// #nosec G115
	if uint32(len(stakerAddr)) < clientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(stakerAddr), clientChainAddrLength)
	}
	delegationParams.StakerAddress = stakerAddr[:clientChainAddrLength]

	// the input operator address is cosmos accAddress type,so we need to check the length and decode it through Bench32
	operatorAddr, ok := args[4].([]byte)
	if !ok || operatorAddr == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 4, "[]byte", args[4])
	}
	if len(operatorAddr) != types.ExoCoreOperatorAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInputOperatorAddrLength, len(operatorAddr), types.ExoCoreOperatorAddrLength)
	}

	opAccAddr, err := sdk.AccAddressFromBech32(string(operatorAddr))
	if err != nil {
		return nil, fmt.Errorf("error occurred when parse acc address from Bech32,the addr is:%s, error:%s", string(operatorAddr), err.Error())
	}
	delegationParams.OperatorAddress = opAccAddr

	opAmount, ok := args[5].(*big.Int)
	if !ok || opAmount == nil || !(opAmount.Cmp(big.NewInt(0)) == 1) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 5, "*big.Int", args[5])
	}
	delegationParams.OpAmount = sdkmath.NewIntFromBigInt(opAmount)
	return delegationParams, nil
}
