package slash

import (
	"fmt"
	"math/big"

	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	"github.com/imua-xyz/imuachain/x/imslash/keeper"
)

func (p Precompile) GetSlashParamsFromInputs(ctx sdk.Context, args []interface{}) (*keeper.SlashParams, error) {
	if len(args) != 8 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}
	slashParams := &keeper.SlashParams{}
	clientChainLzID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "uint32", clientChainLzID)
	}
	slashParams.ClientChainLzID = uint64(clientChainLzID)

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, slashParams.ClientChainLzID)
	if err != nil {
		return nil, err
	}
	clientChainAddrLength := info.AddressLength

	// the length of client chain address inputted by caller is 32, so we need to check the length and remove the padding according to the actual length.
	assetAddr, ok := args[1].([]byte)
	if !ok || assetAddr == nil {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "[]byte", assetAddr)
	}
	// #nosec G115
	if uint32(len(assetAddr)) < clientChainAddrLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidAddrLength, len(assetAddr), clientChainAddrLength)
	}
	slashParams.AssetsAddress = assetAddr[:clientChainAddrLength]

	stakerAddr, ok := args[2].([]byte)
	if !ok || stakerAddr == nil {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "[]byte", stakerAddr)
	}
	// #nosec G115
	if uint32(len(stakerAddr)) < clientChainAddrLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidAddrLength, len(stakerAddr), clientChainAddrLength)
	}
	slashParams.StakerAddress = stakerAddr[:clientChainAddrLength]

	opAmount, ok := args[3].(*big.Int)
	if !ok || opAmount == nil || opAmount.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 3, "*big.Int", opAmount)
	}

	slashParams.OpAmount = sdkmath.NewIntFromBigInt(opAmount)
	return slashParams, nil
}
