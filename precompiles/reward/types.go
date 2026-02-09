package reward

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
	feedistribution "github.com/imua-xyz/imuachain/x/feedistribution/keeper"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

type ClaimRewardArgs struct {
	ClientChainLzID uint32 `abi:"clientChainLzID"`
	StakerAddress   []byte `abi:"stakerAddress"`
}

// WithdrawRewardArgs corresponds to the WithdrawRewardParams struct in Solidity.
// When Solidity encodes a struct parameter, it encodes it as a tuple, which can be
// unpacked directly into this Go struct using method.Inputs.Copy().
type WithdrawRewardArgs struct {
	DoClaim              bool     `abi:"doClaim"`
	ClientChainLzID      uint32   `abi:"clientChainLzID"`
	RewardAssetChainLzID uint32   `abi:"rewardAssetChainLzID"`
	AssetAddress         []byte   `abi:"assetAddress"`
	StakerAddress        []byte   `abi:"stakerAddress"`
	OpAmount             *big.Int `abi:"opAmount"`
}

// WithdrawRewardArgsWrapper wraps WithdrawRewardArgs to match the ABI parameter name.
// This allows method.Inputs.Copy() to work correctly with single tuple parameters.
type WithdrawRewardArgsWrapper struct {
	Params WithdrawRewardArgs `abi:"params"`
}

// WithdrawIMUATokenRewardArgs corresponds to the WithdrawIMUATokenRewardParams struct in Solidity.
// When Solidity encodes a struct parameter, it encodes it as a tuple, which can be
// unpacked directly into this Go struct using method.Inputs.Copy().
type WithdrawIMUATokenRewardArgs struct {
	DoClaim         bool     `abi:"doClaim"`
	ClientChainLzID uint32   `abi:"clientChainLzID"`
	StakerAddress   []byte   `abi:"stakerAddress"`
	ReceiptAddress  []byte   `abi:"receiptAddress"`
	OpAmount        *big.Int `abi:"opAmount"`
}

// WithdrawIMUATokenRewardArgsWrapper wraps WithdrawIMUATokenRewardArgs to match the ABI parameter name.
type WithdrawIMUATokenRewardArgsWrapper struct {
	Params WithdrawIMUATokenRewardArgs `abi:"params"`
}

type SetStakerRewardParamsArgs struct {
	ClientChainLzID    uint32 `abi:"clientChainLzID"`
	StakerAddress      []byte `abi:"stakerAddress"`
	RedelegateReward   bool   `abi:"redelegateReward"`
	RedelegateOperator string `abi:"redelegateOperator"`
}

// UndelegateRewardArgs corresponds to the UndelegateRewardParams struct in Solidity.
// When Solidity encodes a struct parameter, it encodes it as a tuple, which can be
// unpacked directly into this Go struct using method.Inputs.Copy().
type UndelegateRewardArgs struct {
	ClientChainLzID      uint32   `abi:"clientChainLzID"`
	RewardAssetChainLzID uint32   `abi:"rewardAssetChainLzID"`
	AssetAddress         []byte   `abi:"assetAddress"`
	StakerAddress        []byte   `abi:"stakerAddress"`
	OperatorAddr         string   `abi:"operatorAddr"`
	OpAmount             *big.Int `abi:"opAmount"`
	InstantUnbond        bool     `abi:"instantUnbond"`
}

// UndelegateRewardArgsWrapper wraps UndelegateRewardArgs to match the ABI parameter name.
type UndelegateRewardArgsWrapper struct {
	Params UndelegateRewardArgs `abi:"params"`
}

type WithdrawCommissionArgs struct {
	RewardAssetChainLzID uint32   `abi:"rewardAssetChainLzID"`
	AssetAddress         []byte   `abi:"assetAddress"`
	OperatorAddress      []byte   `abi:"operatorAddress"`
	OpAmount             *big.Int `abi:"opAmount"`
}

type WithdrawIMUATokenCommissionArgs struct {
	OperatorAddress []byte   `abi:"operatorAddress"`
	ReceiptAddress  []byte   `abi:"receiptAddress"`
	OpAmount        *big.Int `abi:"opAmount"`
}

// RegisterRewardTokenArgs corresponds to the RegisterRewardTokenParams struct in Solidity.
// When Solidity encodes a struct parameter, it encodes it as a tuple, which can be
// unpacked directly into this Go struct using method.Inputs.Copy().
type RegisterRewardTokenArgs struct {
	ClientChainID        uint32 `abi:"clientChainID"`
	Token                []byte `abi:"token"`
	Decimals             uint8  `abi:"decimals"`
	Name                 string `abi:"name"`
	Symbol               string `abi:"symbol"`
	MetaData             string `abi:"metaData"`
	Denomination         string `abi:"denomination"`
	DenominationExponent uint8  `abi:"denominationExponent"`
}

// RegisterRewardTokenArgsWrapper wraps RegisterRewardTokenArgs to match the ABI parameter name.
type RegisterRewardTokenArgsWrapper struct {
	Params RegisterRewardTokenArgs `abi:"params"`
}

type UpdateRewardTokenArgs struct {
	ClientChainID uint32 `abi:"clientChainID"`
	Token         []byte `abi:"token"`
	MetaData      string `abi:"metaData"`
}

type ABIRewardCoin struct {
	Denomination string   `abi:"denomination"`
	Amount       *big.Int `abi:"amount"`
}

type ABIOperatorRewardProportion struct {
	Operator    string   `abi:"operator"`
	Numerator   *big.Int `abi:"numerator"`
	Denominator *big.Int `abi:"denominator"`
}

type AVSRewardDistributionInfoArgs struct {
	RewardCoins               []ABIRewardCoin               `abi:"rewardCoins"`
	OperatorRewardProportions []ABIOperatorRewardProportion `abi:"operatorRewardProportions"`
}

// AVSRewardDistributionInfoArgsWrapper wraps AVSRewardDistributionInfoArgs to match the ABI parameter name.
type AVSRewardDistributionInfoArgsWrapper struct {
	RewardDistribution AVSRewardDistributionInfoArgs `abi:"rewardDistribution"`
}

type SetAVSEpochRewardArgs struct {
	EpochRewards []ABIRewardCoin `abi:"epochRewards"`
}

type SetOperatorRewardProportionsArgs struct {
	OperatorRewardProportions []ABIOperatorRewardProportion `abi:"operatorRewardProportions"`
}

type SetAVSRewardParamsArgs struct {
	IsCustomRewardInflation bool `abi:"isCustomRewardInflation"`
	IsCustomOperatorRatio   bool `abi:"isCustomOperatorRatio"`
}

type FundAVSRewardArgs struct {
	RewardAssetChainLzID uint32         `abi:"rewardAssetChainLzID"`
	AVSAddress           common.Address `abi:"avsAddress"`
	AssetAddress         []byte         `abi:"assetAddress"`
	OpAmount             *big.Int       `abi:"opAmount"`
}

type IsRegisterRewardTokenArgs struct {
	ClientChainID uint32 `abi:"clientChainID"`
	Token         []byte `abi:"token"`
}

type ABIRewardCoins []ABIRewardCoin

type ABIOperatorRewardProportions []ABIOperatorRewardProportion

func (ar ABIRewardCoins) ToProtoStruct(ctx sdk.Context, avsAddr string, k feedistribution.Keeper) (sdk.DecCoins, error) {
	ret := make([]sdk.DecCoin, 0)
	validationFunc := func(_ int, rewardCoin ABIRewardCoin) error {
		if rewardCoin.Amount == nil || !(rewardCoin.Amount.Cmp(big.NewInt(0)) == 1) {
			return fmt.Errorf("ABIRewardCoins.ToProtoStruct: invalid amount:%v", rewardCoin.Amount)
		}
		// get the reward asset decimal
		_, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, avsAddr, rewardCoin.Denomination)
		if err != nil {
			return err
		}
		amountDecimal := feedistributiontypes.ScaleIntByDecimals(sdkmath.NewIntFromBigInt(rewardCoin.Amount), rewardAsset.RewardAssetInfo.DenominationExponent)
		if amountDecimal.IsNil() || !amountDecimal.IsPositive() {
			return fmt.Errorf("ABIRewardCoins.ToProtoStruct: invalid amount after converting to devimal:%s", amountDecimal)
		}
		ret = append(ret, sdk.DecCoin{
			Denom:  rewardCoin.Denomination,
			Amount: amountDecimal,
		})
		return nil
	}
	seenFieldValueFunc := func(rewardCoin ABIRewardCoin) (string, struct{}) {
		return rewardCoin.Denomination, struct{}{}
	}
	_, err := utils.CommonValidation(ar, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, err
	}
	// the coins should be sorted.
	return sdk.DecCoins(ret).Sort(), nil
}

func (ap ABIOperatorRewardProportions) ToProtoStruct() ([]feedistributiontypes.OperatorRewardProportion, error) {
	ret := make([]feedistributiontypes.OperatorRewardProportion, 0)
	totalProportion := sdkmath.LegacyZeroDec()
	validationFunc := func(_ int, operatorRewardProportion ABIOperatorRewardProportion) error {
		if operatorRewardProportion.Numerator == nil || !(operatorRewardProportion.Numerator.Cmp(big.NewInt(0)) == 1) {
			return fmt.Errorf("ABIOperatorRewardProportions.ToProtoStruct: invalid numerator:%v",
				operatorRewardProportion.Numerator)
		}
		if operatorRewardProportion.Denominator == nil || !(operatorRewardProportion.Denominator.Cmp(big.NewInt(0)) == 1) {
			return fmt.Errorf("ABIOperatorRewardProportions.ToProtoStruct: invalid denominator:%v",
				operatorRewardProportion.Denominator)
		}
		if operatorRewardProportion.Numerator.Cmp(operatorRewardProportion.Denominator) == 1 {
			return fmt.Errorf("ABIOperatorRewardProportions.ToProtoStruct: numerator shouldn't be greater than the denominator, numerator:%s,denominator:%s", operatorRewardProportion.Numerator, operatorRewardProportion.Denominator)
		}
		_, err := sdk.AccAddressFromBech32(operatorRewardProportion.Operator)
		if err != nil {
			return fmt.Errorf("ABIOperatorRewardProportions.ToProtoStruct: invalid opeartor address,the addr is:%s, error:%s", operatorRewardProportion.Operator, err.Error())
		}
		proportion := sdk.NewDecFromBigInt(operatorRewardProportion.Numerator).Quo(sdk.NewDecFromBigInt(operatorRewardProportion.Denominator))

		ret = append(ret, feedistributiontypes.OperatorRewardProportion{
			OperatorAddr:     operatorRewardProportion.Operator,
			RewardProportion: proportion,
		})
		totalProportion.AddMut(proportion)
		if totalProportion.GT(sdkmath.LegacyNewDec(1)) {
			return fmt.Errorf("ABIOperatorRewardProportions.ToProtoStruct: total reward proportion shouldn't be greater than 1, totalProportion:%s", totalProportion)
		}
		return nil
	}
	seenFieldValueFunc := func(operatorRewardProportion ABIOperatorRewardProportion) (string, struct{}) {
		return operatorRewardProportion.Operator, struct{}{}
	}
	_, err := utils.CommonValidation(ap, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
