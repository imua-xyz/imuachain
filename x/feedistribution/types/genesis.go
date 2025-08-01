package types

import (
	"strings"

	avstypes "github.com/imua-xyz/imuachain/x/avs/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	"golang.org/x/xerrors"
)

type KeyTypeForJoinedKey int

const (
	StakerID KeyTypeForJoinedKey = iota
	AssetID
	OperatorAddr
	AVSAddr
	EpochIdentifier
	PeriodHexStr
	EpochNumberHexStr
	BlockHeightHexStr
)

var IMUARewardToken = AVSRewardAsset{
	AssetBasicInfo: assetstypes.AssetInfo{
		Name:             "Native IM token",
		Symbol:           utils.BaseDenom,
		Address:          "0x0000000000000000000000000000000000000000",
		Decimals:         0,
		LayerZeroChainID: 0,
		MetaInfo:         "IMUA native to Imuachain",
	},
	RewardAssetState: AVSRewardAssetState{
		RewardPoolBalance:     sdk.ZeroDec(),
		RewardPoolTotal:       sdk.ZeroDec(),
		RewardAllocationTotal: sdk.ZeroDec(),
	},
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	// Use the default chain ID to generate the dogfood address, the current default genesis is used for mainnet.
	// The AVS address in the genesis file should be updated either manually or via a script when used for testnet.
	avsAddrStr := avstypes.GenerateAVSAddress(avstypes.ChainIDWithoutRevision(utils.DefaultChainID))
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
		AllAvsRewardAssets: []AVSAddrAndRewardAssets{
			{
				Avs: avsAddrStr,
				AvsRewardAssets: []AVSRewardAsset{
					IMUARewardToken,
				},
			},
		},
	}
}

func NewGenesisState(p Params) *GenesisState {
	return &GenesisState{
		Params: p,
	}
}

func CheckEVMHexAddress(s string, checkLowercase bool) bool {
	if checkLowercase && strings.ToLower(s) != s {
		return false
	}
	return common.IsHexAddress(s)
}

func CheckUint64BigEndianHexStr(s string) error {
	if strings.ToLower(s) != s {
		return xerrors.Errorf("the period hex string should be in lowercase.")
	}
	bytes, err := hexutil.Decode(s)
	if err != nil {
		return err
	}
	// check if the length of bytes is 8.
	if len(bytes) != 8 {
		return xerrors.Errorf("the length of period bytes should be 8.")
	}
	return nil
}

func CheckJoinedKey(joinedKey string, keyNumber int, keyTypes []KeyTypeForJoinedKey) error {
	keys, err := assetstypes.ParseJoinedStoreKey([]byte(joinedKey), keyNumber)
	if err != nil {
		return xerrors.Errorf("failed to parse the key, err:%s,key:%s", err, joinedKey)
	}
	if len(keyTypes) != keyNumber {
		return xerrors.Errorf("the length of key types slice isn't equal to the input key number, keyNumber:%d,len(keyTypes):%d", keyNumber, len(keyTypes))
	}
	for index, keyType := range keyTypes {
		switch keyType {
		case StakerID:
			if _, _, err := assetstypes.ValidateID(keys[index], true, false); err != nil {
				return xerrors.Errorf("invalid stakerID, key:%s,stakerID:%s", joinedKey, keys[index])
			}
		case AssetID:
			if _, _, err := assetstypes.ValidateID(keys[index], true, false); err != nil {
				return xerrors.Errorf("invalid assetID, key:%s,assetID:%s", joinedKey, keys[index])
			}
		case OperatorAddr:
			if _, err := sdk.AccAddressFromBech32(keys[index]); err != nil {
				return xerrors.Errorf(
					"invalid operator address, operator:%s, key:%s", keys[index], joinedKey,
				)
			}
		case AVSAddr:
			if !CheckEVMHexAddress(keys[index], true) {
				return xerrors.Errorf("invalid avs address, avsAddr:%s, key:%s", keys[index], joinedKey)
			}
		case EpochIdentifier:
			err = epochstypes.ValidateEpochIdentifierString(keys[index])
			if err != nil {
				return xerrors.Errorf("invalid epoch identifier, err:%s,identifier:%s, key:%s", err, keys[index], joinedKey)
			}
		case PeriodHexStr:
			err = CheckUint64BigEndianHexStr(keys[index])
			if err != nil {
				return xerrors.Errorf("invalid period, err:%s,periodHexStr:%s, key:%s", err, keys[index], joinedKey)
			}
		case EpochNumberHexStr:
			err = CheckUint64BigEndianHexStr(keys[index])
			if err != nil {
				return xerrors.Errorf("invalid epoch number, err:%s,epochNumberHexStr:%s, key:%s", err, keys[index], joinedKey)
			}
		case BlockHeightHexStr:
			err = CheckUint64BigEndianHexStr(keys[index])
			if err != nil {
				return xerrors.Errorf("invalid block height, err:%s,BlockHeightHexStr:%s, key:%s", err, keys[index], joinedKey)
			}
		}
	}
	return nil
}

func (gs GenesisState) ValidateAVSRewardAssets() error {
	validationFunc := func(_ int, info AVSAddrAndRewardAssets) error {
		// check the avs address
		if !CheckEVMHexAddress(info.Avs, true) {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSRewardAssets: invalid avs address, avsAddr:%s", info.Avs)
		}

		// check the reward assets list
		seenFieldValueFunc := func(rewardAssetInfo AVSRewardAsset) (string, struct{}) {
			_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
				rewardAssetInfo.AssetBasicInfo.LayerZeroChainID,
				"", rewardAssetInfo.AssetBasicInfo.Address,
			)
			return assetID, struct{}{}
		}
		validationFunc := func(_ int, rewardAssetInfo AVSRewardAsset) error {
			address := rewardAssetInfo.AssetBasicInfo.Address
			if strings.ToLower(address) != address {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateAVSRewardAssets: contains uppercase characters for reward token %s, address: %s,avsAddr:%s", rewardAssetInfo.AssetBasicInfo.Name, rewardAssetInfo.AssetBasicInfo.Address, info.Avs,
				)
			}
			// check the decimal
			if rewardAssetInfo.AssetBasicInfo.Decimals > assetstypes.MaxDecimal {
				return ErrInvalidGenesisData.Wrapf("the decimal is greater than the MaxDecimal,decimal:%v,MaxDecimal:%v", rewardAssetInfo.AssetBasicInfo.Decimals, assetstypes.MaxDecimal)
			}
			// check the symbol
			err := sdk.ValidateDenom(rewardAssetInfo.AssetBasicInfo.Symbol)
			if err != nil {
				return ErrInvalidGenesisData.Wrapf("symbol should be a valid denomination,symbol:%s,err:%s", rewardAssetInfo.AssetBasicInfo.Symbol, err)
			}
			if rewardAssetInfo.RewardAssetState.RewardPoolBalance.IsNil() ||
				rewardAssetInfo.RewardAssetState.RewardPoolBalance.IsNegative() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"ValidateAVSRewardAssets: nil or negative reward pool balance ,avs:%s,rewardAssetName:%s",
					info.Avs, rewardAssetInfo.AssetBasicInfo.Name,
				)
			}
			if rewardAssetInfo.RewardAssetState.RewardPoolTotal.IsNil() ||
				rewardAssetInfo.RewardAssetState.RewardPoolTotal.IsNegative() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"ValidateAVSRewardAssets: nil or negative reward pool total amount ,avs:%s,rewardAssetName:%s",
					info.Avs, rewardAssetInfo.AssetBasicInfo.Name,
				)
			}
			if rewardAssetInfo.RewardAssetState.RewardPoolBalance.GT(rewardAssetInfo.RewardAssetState.RewardPoolTotal) {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"ValidateAVSRewardAssets: reward pool balance is great than the reward pool total amount ,avs:%s,rewardAssetName:%s",
					info.Avs, rewardAssetInfo.AssetBasicInfo.Name,
				)
			}
			if rewardAssetInfo.RewardAssetState.RewardAllocationTotal.IsNil() ||
				rewardAssetInfo.RewardAssetState.RewardAllocationTotal.IsNegative() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"ValidateAVSRewardAssets: nil or negative reward allocation total amount ,avs:%s,rewardAssetName:%s",
					info.Avs, rewardAssetInfo.AssetBasicInfo.Name,
				)
			}
			return nil
		}
		_, err := utils.CommonValidation(info.AvsRewardAssets, seenFieldValueFunc, validationFunc)
		if err != nil {
			return err
		}
		return nil
	}
	seenFieldValueFunc := func(info AVSAddrAndRewardAssets) (string, struct{}) {
		return info.Avs, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllAvsRewardAssets, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateAVSRewardParams() error {
	validationFunc := func(_ int, info AVSAddrAndRewardParam) error {
		// check the avs address
		// Case sensitivity is not a concern here, as the AVS address is decoded into bytes
		// before being used as the key in the store
		if !common.IsHexAddress(info.Avs) {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSRewardParams: invalid avs address, avsAddr:%s", info.Avs)
		}
		return nil
	}
	seenFieldValueFunc := func(info AVSAddrAndRewardParam) (string, struct{}) {
		return info.Avs, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllAvsRewardParams, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateAVSFeePools() error {
	validationFunc := func(_ int, info AVSAddrAndFeePool) error {
		// check the avs address
		// Case sensitivity is not a concern here, as the AVS address is decoded into bytes
		// before being used as the key in the store
		if !common.IsHexAddress(info.Avs) {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSFeePools: invalid avs address, avsAddr:%s", info.Avs)
		}
		if !info.AvsFeePool.CommunityPool.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSFeePools: invalid community pool, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. avsAddr:%s", info.Avs)
		}
		return nil
	}
	seenFieldValueFunc := func(info AVSAddrAndFeePool) (string, struct{}) {
		return info.Avs, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllAvsFeePools, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateAVSRewardDistributions() error {
	validationFunc := func(_ int, info AVSAddrAndRewardDistribution) error {
		totalProportion := sdkmath.LegacyZeroDec()
		// check the avs address
		// Case sensitivity is not a concern here, as the AVS address is decoded into bytes
		// before being used as the key in the store
		if !common.IsHexAddress(info.Avs) {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSRewardDistributions: invalid avs address, avsAddr:%s", info.Avs)
		}
		if !info.AvsRewardDistribution.Rewards.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSRewardDistributions: invalid rewards, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. avsAddr:%s", info.Avs)
		}

		if info.AvsRewardDistribution.RewardsEpochNumber < 0 || info.AvsRewardDistribution.ProportionsEpochNumber < 0 {
			return ErrInvalidGenesisData.Wrapf("ValidateAVSRewardDistributions: negative epoch number,RewardsEpochNumber:%d,ProportionsEpochNumber:%d",
				info.AvsRewardDistribution.RewardsEpochNumber, info.AvsRewardDistribution.ProportionsEpochNumber)
		}

		// check the operator proportions
		validationFunc := func(_ int, proportion OperatorRewardProportion) error {
			if _, err := sdk.AccAddressFromBech32(proportion.OperatorAddr); err != nil {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateAVSRewardDistributions: invalid operator address,avs:%s, operator: %s, err: %s", info.Avs, proportion.OperatorAddr, err,
				)
			}
			if proportion.RewardProportion.IsNil() ||
				proportion.RewardProportion.LTE(sdkmath.LegacyZeroDec()) ||
				proportion.RewardProportion.GT(sdkmath.LegacyNewDec(1)) {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateAVSRewardDistributions: nil or negative reward proportion ,avs:%s,operator:%s",
					info.Avs, proportion.OperatorAddr,
				)
			}
			totalProportion.AddMut(proportion.RewardProportion)
			return nil
		}
		seenFieldValueFunc := func(proportion OperatorRewardProportion) (string, struct{}) {
			return proportion.OperatorAddr, struct{}{}
		}
		_, err := utils.CommonValidation(info.AvsRewardDistribution.OperatorRewardProportions,
			seenFieldValueFunc, validationFunc)
		if err != nil {
			return err
		}
		if totalProportion.GT(sdkmath.LegacyNewDec(1)) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAVSRewardDistributions: total operator proportions > 1 for avs %s: %s",
				info.Avs, totalProportion,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(info AVSAddrAndRewardDistribution) (string, struct{}) {
		return info.Avs, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllAvsRewardDistributions, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorOutstandingRewards() error {
	validationFunc := func(_ int, info KeyAndOperatorOutstandingRewards) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 2, []KeyTypeForJoinedKey{OperatorAddr, AVSAddr})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorOutstandingRewards: failed to check the joined key, err:%s", err)
		}
		if !info.OperatorOutstandingRewards.Rewards.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorOutstandingRewards: invalid outstanding reward for operator, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. key:%s",
				info.Key)
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndOperatorOutstandingRewards) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllOperatorOutstandingRewards, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateDelegationChangeInfos() error {
	validationFunc := func(_ int, info KeyAndDelegationChangeInfo) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 3, []KeyTypeForJoinedKey{EpochIdentifier, OperatorAddr, AssetID})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateDelegationChangeInfos: failed to check the joined key, err:%s", err)
		}
		// check the total amount in the delegation change info
		if info.DelegationChangeInfo.TotalAmount.IsNil() ||
			info.DelegationChangeInfo.TotalAmount.IsNegative() {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"ValidateDelegationChangeInfos: nil or negative total amount in the delegation change info ,key:%s,totalAmount:%v", info.Key, info.DelegationChangeInfo.TotalAmount,
			)
		}
		// check the staker list
		validationFunc := func(_ int, delegationChangeInfo StakerDelegationChange) error {
			if _, _, err := assetstypes.ValidateID(delegationChangeInfo.StakerId, true, false); err != nil {
				return ErrInvalidGenesisData.Wrapf("ValidateDelegationChangeInfos: invalid stakerID, key:%s,stakerID:%s", info.Key, delegationChangeInfo.StakerId)
			}
			if delegationChangeInfo.PreviousDelegatedAmount.IsNil() ||
				delegationChangeInfo.PreviousDelegatedAmount.IsNegative() {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"ValidateDelegationChangeInfos: nil or negative PreviousDelegatedAmount,stakerID:%s,PreviousDelegatedAmount:%v", delegationChangeInfo.StakerId, delegationChangeInfo.PreviousDelegatedAmount,
				)
			}
			return nil
		}
		seenFieldValueFunc := func(delegationChangeInfo StakerDelegationChange) (string, struct{}) {
			return delegationChangeInfo.StakerId, struct{}{}
		}
		_, err = utils.CommonValidation(info.DelegationChangeInfo.StakerDelegationChanges, seenFieldValueFunc, validationFunc)
		if err != nil {
			return err
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndDelegationChangeInfo) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllDelegationChangeInfos, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateDelegationStartingInfos() error {
	validationFunc := func(_ int, info KeyAndDelegationStartingInfo) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 4,
			[]KeyTypeForJoinedKey{StakerID, AssetID, OperatorAddr, EpochIdentifier})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateDelegationStartingInfos: failed to check the joined key, err:%s", err)
		}
		// check the stake in the delegation starting info
		if info.DelegationStartingInfo.Stake.IsNil() ||
			info.DelegationStartingInfo.Stake.IsNegative() {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"ValidateDelegationStartingInfos: nil or negative stake in the delegation starting info ,key:%s,totalAmount:%v", info.Key, info.DelegationStartingInfo.Stake,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndDelegationStartingInfo) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllDelegationStartingInfos, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorHistoricalRewards() error {
	validationFunc := func(_ int, info KeyAndOperatorHistoricalRewards) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 4,
			[]KeyTypeForJoinedKey{OperatorAddr, AssetID, EpochIdentifier, PeriodHexStr})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorHistoricalRewards: failed to check the joined key, err:%s", err)
		}
		// check the reference count
		if info.OperatorHistoricalRewards.ReferenceCount == 0 {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorHistoricalRewards: reference count shouldn't be zero, key:%s", info.Key)
		}
		// check the historical reward ratios for an operator
		validationFunc := func(_ int, rewards CommonAVSRewardData) error {
			// check the avs address
			if !CheckEVMHexAddress(rewards.AVSAddress, true) {
				return ErrInvalidGenesisData.Wrapf("ValidateOperatorHistoricalRewards: invalid avs address, avsAddr:%s,key:%s", rewards.AVSAddress, info.Key)
			}
			if !rewards.Rewards.IsValid() {
				return ErrInvalidGenesisData.Wrapf("ValidateOperatorHistoricalRewards: invalid historical reward for specific operator and avs, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens, avsAddr:%s,key:%s", rewards.AVSAddress, info.Key)
			}
			return nil
		}
		seenFieldValueFunc := func(rewards CommonAVSRewardData) (string, struct{}) {
			return rewards.AVSAddress, struct{}{}
		}
		_, err = utils.CommonValidation(info.OperatorHistoricalRewards.CumulativeRewardRatios, seenFieldValueFunc, validationFunc)
		if err != nil {
			return err
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndOperatorHistoricalRewards) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllOperatorHistoricalRewards, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorCurrentRewards() error {
	validationFunc := func(_ int, info KeyAndOperatorCurrentRewards) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 3,
			[]KeyTypeForJoinedKey{OperatorAddr, AssetID, EpochIdentifier})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorCurrentRewards: failed to check the joined key, err:%s", err)
		}
		// check the period in the current rewards
		if info.OperatorCurrentRewards.Period == 0 {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorCurrentRewards: period shouldn't be zero, key:%s", info.Key)
		}
		// check the historical reward ratios for an operator
		validationFunc := func(_ int, rewards CommonAVSRewardData) error {
			// check the avs address
			if !CheckEVMHexAddress(rewards.AVSAddress, true) {
				return ErrInvalidGenesisData.Wrapf("ValidateOperatorCurrentRewards: invalid avs address, avsAddr:%s,key:%s", rewards.AVSAddress, info.Key)
			}
			if !rewards.Rewards.IsValid() {
				return ErrInvalidGenesisData.Wrapf("ValidateOperatorCurrentRewards: invalid historical reward for specific operator and avs, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens, avsAddr:%s,key:%s", rewards.AVSAddress, info.Key)
			}
			return nil
		}
		seenFieldValueFunc := func(rewards CommonAVSRewardData) (string, struct{}) {
			return rewards.AVSAddress, struct{}{}
		}
		_, err = utils.CommonValidation(info.OperatorCurrentRewards.Rewards, seenFieldValueFunc, validationFunc)
		if err != nil {
			return err
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndOperatorCurrentRewards) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllOperatorCurrentRewards, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorCommissions() error {
	validationFunc := func(_ int, info KeyAndOperatorCommission) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 2,
			[]KeyTypeForJoinedKey{OperatorAddr, AVSAddr})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorCommissions: failed to check the joined key, err:%s", err)
		}
		if !info.OperatorCommission.UnwithdrawnCommission.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorCommissions: invalid unwithdrawn commission for operator, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. key:%s", info.Key)
		}
		if !info.OperatorCommission.WithdrawnCommission.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorCommissions: invalid withdrawn commission for operator, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. key:%s", info.Key)
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndOperatorCommission) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllOperatorCommission, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorSlashEvents() error {
	validationFunc := func(_ int, info KeyAndOperatorSlashEvent) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 5,
			[]KeyTypeForJoinedKey{OperatorAddr, AssetID, EpochIdentifier, EpochNumberHexStr, BlockHeightHexStr})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorSlashEvents: failed to check the joined key, err:%s", err)
		}
		if info.OperatorSlashEvent.Fraction.IsNil() ||
			info.OperatorSlashEvent.Fraction.LTE(sdkmath.LegacyZeroDec()) ||
			info.OperatorSlashEvent.Fraction.GT(sdkmath.LegacyNewDec(1)) {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"ValidateOperatorSlashEvents: nil or invalid slash fraction,key:%s,fraction:%v", info.Key, info.OperatorSlashEvent.Fraction,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndOperatorSlashEvent) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllOperatorSlashEvents, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateClaimedOutstandingRewards() error {
	validationFunc := func(_ int, info KeyAndStakerClaimedRewards) error {
		// check the joined key
		err := CheckJoinedKey(info.Key, 2, []KeyTypeForJoinedKey{StakerID, AVSAddr})
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateClaimedOutstandingRewards: failed to check the joined key, err:%s", err)
		}
		if !info.StakerClaimedRewards.OutstandingRewards.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateClaimedOutstandingRewards: invalid outstanding reward for staker, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. key:%s",
				info.Key)
		}
		if !info.StakerClaimedRewards.WithdrawnRewards.IsValid() {
			return ErrInvalidGenesisData.Wrapf("ValidateClaimedOutstandingRewards: invalid withdrawn reward for staker, DecCoins are unsorted, contain negative amounts, or have duplicate reward tokens. key:%s",
				info.Key)
		}
		return nil
	}
	seenFieldValueFunc := func(info KeyAndStakerClaimedRewards) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AllStakerClaimedRewards, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return err
	}
	err = gs.ValidateAVSRewardAssets()
	if err != nil {
		return err
	}
	err = gs.ValidateAVSRewardParams()
	if err != nil {
		return err
	}
	err = gs.ValidateAVSFeePools()
	if err != nil {
		return err
	}
	err = gs.ValidateAVSRewardDistributions()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorOutstandingRewards()
	if err != nil {
		return err
	}
	err = gs.ValidateDelegationChangeInfos()
	if err != nil {
		return err
	}
	err = gs.ValidateDelegationStartingInfos()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorHistoricalRewards()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorCurrentRewards()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorCommissions()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorSlashEvents()
	if err != nil {
		return err
	}
	err = gs.ValidateClaimedOutstandingRewards()
	if err != nil {
		return err
	}
	return nil
}
