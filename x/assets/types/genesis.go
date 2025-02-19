package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
)

// NewGenesis returns a new genesis state with the given inputs.
func NewGenesis(
	params Params, chains []ClientChainInfo,
	tokens []StakingAssetInfo, deposits []DepositsByStaker,
	operatorAssets []AssetsByOperator,
) *GenesisState {
	return &GenesisState{
		Params:         params,
		ClientChains:   chains,
		Tokens:         tokens,
		Deposits:       deposits,
		OperatorAssets: operatorAssets,
	}
}

// DefaultGenesis returns the default genesis state. It intentionally
// does not have any supported assets or deposits, since these must
// be supplied manually before bootstrapping the chain. The same is true
// for any unit / integration tests.
func DefaultGenesis() *GenesisState {
	return NewGenesis(
		DefaultParams(), []ClientChainInfo{}, []StakingAssetInfo{}, []DepositsByStaker{}, []AssetsByOperator{},
	)
}

// ValidateClientChains performs basic client chains validation
func (gs GenesisState) ValidateClientChains() (map[uint64]struct{}, error) {
	validationFunc := func(i int, info ClientChainInfo) error {
		if info.Name == "" {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"nil Name for chain %d",
				i,
			)
		}
		// the address length is used to convert from bytes32 to address.
		if info.AddressLength == 0 {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"nil AddressLength for chain %s",
				info.Name,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(info ClientChainInfo) (uint64, struct{}) {
		return info.LayerZeroChainID, struct{}{}
	}
	lzIDs, err := utils.CommonValidation(gs.ClientChains, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}

	return lzIDs, nil
}

// ValidateTokens performs basic client chain assets validation
func (gs GenesisState) ValidateTokens(lzIDs map[uint64]struct{}) (map[string]math.Int, error) {
	validationFunc := func(_ int, info StakingAssetInfo) error {
		id := info.AssetBasicInfo.LayerZeroChainID
		// check that the chain is registered
		if _, ok := lzIDs[id]; !ok {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"unknown LayerZeroChainID for token %s, clientChainID: %d",
				info.AssetBasicInfo.MetaInfo, id,
			)
		}
		address := info.AssetBasicInfo.Address
		if strings.ToLower(address) != address {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"contains uppercase characters for token %s, address: %s",
				info.AssetBasicInfo.Name, address,
			)
		}

		// the initial deposit quantity should be non-negative.
		// ideally, we should verify that this quantity sums
		// each staker's deposit, but that is not possible solely inside
		// this module.
		// deposit = free + pending undelegation + delegated, of which
		// the last is not stored in this module.
		if info.StakingTotalAmount.IsNil() ||
			info.StakingTotalAmount.IsNegative() {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"nil total staking amount for asset %s",
				info.AssetBasicInfo.Address,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(info StakingAssetInfo) (string, math.Int) {
		// calculate the asset id.
		_, assetID := GetStakerIDAndAssetIDFromStr(
			info.AssetBasicInfo.LayerZeroChainID,
			"", info.AssetBasicInfo.Address,
		)
		return assetID, info.StakingTotalAmount
	}
	totalStaking, err := utils.CommonValidation(gs.Tokens, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}
	return totalStaking, nil
}

// ValidateDeposits performs basic deposits validation
func (gs GenesisState) ValidateDeposits(lzIDs map[uint64]struct{}, tokensTotalStaking map[string]math.Int) error {
	validationFunc := func(_ int, depositByStaker DepositsByStaker) error {
		stakerID := depositByStaker.StakerID
		// validate the stakerID
		var stakerClientChainID uint64
		var err error
		if _, stakerClientChainID, err = ValidateID(stakerID, true, true); err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"invalid stakerID: %s",
				stakerID,
			)
		}
		// check that the chain is registered
		if _, ok := lzIDs[stakerClientChainID]; !ok {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"unknown LayerZeroChainID for staker %s: %d",
				stakerID, stakerClientChainID,
			)
		}

		// validate the deposits
		depositValidationF := func(_ int, deposit DepositByAsset) error {
			assetID := deposit.AssetID
			// check that the asset is registered
			// no need to check for the validity of the assetID, since
			// an invalid assetID cannot be in the tokens map.
			tokenTotalStaking, ok := tokensTotalStaking[assetID]
			if !ok {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"unknown assetID for deposit %s: %s",
					stakerID, assetID,
				)
			}
			// #nosec G703 // if it's invalid, we will not reach here.
			_, assetClientChainID, _ := ParseID(assetID)
			if assetClientChainID != stakerClientChainID {
				// we can reach here if there are multiple chains
				// and it tries to deposit assets from one chain
				// under a staker from another chain.
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"mismatched client chain IDs for staker %s and asset %s",
					stakerID, assetID,
				)
			}

			info := deposit.Info
			// check that there is no nil value provided.
			if info.TotalDepositAmount.IsNil() || info.WithdrawableAmount.IsNil() ||
				info.PendingUndelegationAmount.IsNil() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"nil deposit info for %s: %+v",
					assetID, info,
				)
			}

			// check for negative values.
			if info.TotalDepositAmount.IsNegative() || info.WithdrawableAmount.IsNegative() ||
				info.PendingUndelegationAmount.IsNegative() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"negative deposit amount for %s: %+v",
					assetID, info,
				)
			}

			if info.TotalDepositAmount.GT(tokenTotalStaking) {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"invalid deposit amount that is greater than the total staking, assetID: %s: %+v",
					assetID, info,
				)
			}

			if info.PendingUndelegationAmount.Add(info.WithdrawableAmount).GT(info.TotalDepositAmount) {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"the sum of PendingUndelegationAmount and WithdrawableAmount is greater than the TotalDepositAmount, assetID: %s: %+v",
					assetID, info,
				)
			}
			return nil
		}
		depositFieldValueF := func(deposit DepositByAsset) (string, struct{}) {
			return deposit.AssetID, struct{}{}
		}
		_, err = utils.CommonValidation(depositByStaker.Deposits, depositFieldValueF, depositValidationF)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}
		return nil
	}
	seenFieldValueFunc := func(deposits DepositsByStaker) (string, struct{}) {
		return deposits.StakerID, struct{}{}
	}
	_, err := utils.CommonValidation(gs.Deposits, seenFieldValueFunc, validationFunc)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}
	return nil
}

// ValidateOperatorAssets performs basic operator assets validation
func (gs GenesisState) ValidateOperatorAssets(tokensTotalStaking map[string]math.Int) error {
	validationFunc := func(_ int, assets AssetsByOperator) error {
		_, err := sdk.AccAddressFromBech32(assets.Operator)
		if err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"invalid operator address %s: %s", assets.Operator, err,
			)
		}

		// validate the assets list for the specified operator
		assetValidationFunc := func(_ int, asset AssetByID) error {
			// check that the asset is registered
			// no need to check for the validity of the assetID, since
			// an invalid assetID cannot be in the tokens map.
			totalStaking, ok := tokensTotalStaking[asset.AssetID]
			if !ok {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"unknown assetID for operator assets %s: %s",
					assets.Operator, asset.AssetID,
				)
			}
			// the sum amount of operators shouldn't be greater than the total staking amount of this asset.
			// however, this line should not apply for the native asset, since that is handled by x/bank.
			if asset.Info.TotalAmount.Add(asset.Info.PendingUndelegationAmount).GT(totalStaking) &&
				asset.AssetID != ImuachainAssetID {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"operator's sum amount exceeds the total staking amount for %s: %+v",
					assets.Operator, asset,
				)
			}

			if asset.Info.OperatorShare.GT(asset.Info.TotalShare) {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"operator's share exceeds the total share for %s: %+v",
					assets.Operator, asset,
				)
			}
			return nil
		}
		assetFieldValueFunc := func(asset AssetByID) (string, struct{}) {
			return asset.AssetID, struct{}{}
		}
		_, err = utils.CommonValidation(assets.AssetsState, assetFieldValueFunc, assetValidationFunc)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}
		return nil
	}
	seenFieldValueFunc := func(assets AssetsByOperator) (string, struct{}) {
		return assets.Operator, struct{}{}
	}
	_, err := utils.CommonValidation(gs.OperatorAssets, seenFieldValueFunc, validationFunc)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}
	return nil
}

// Validate performs basic genesis state validation returning an error
// upon any failure.
func (gs GenesisState) Validate() error {
	lzIDs, err := gs.ValidateClientChains()
	if err != nil {
		return err
	}
	totalStaking, err := gs.ValidateTokens(lzIDs)
	if err != nil {
		return err
	}
	err = gs.ValidateDeposits(lzIDs, totalStaking)
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorAssets(totalStaking)
	if err != nil {
		return err
	}
	return gs.Params.Validate()
}
