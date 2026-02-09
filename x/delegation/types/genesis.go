package types

import (
	"encoding/hex"

	epochsTypes "github.com/imua-xyz/imuachain/x/epochs/types"

	"github.com/imua-xyz/imuachain/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"golang.org/x/xerrors"
)

// NewGenesis returns a new genesis state with the given inputs.
func NewGenesis(
	params Params,
	associations []StakerToOperator,
	delegationStates []DelegationStates,
	stakersByOperator []string,
	undelegations []UndelegationAndHoldCount,
) *GenesisState {
	return &GenesisState{
		Params:            params,
		Associations:      associations,
		DelegationStates:  delegationStates,
		StakersByOperator: stakersByOperator,
		Undelegations:     undelegations,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return NewGenesis(DefaultParams(), nil, nil, nil, nil)
}

func ValidateIDAndOperator(stakerID, assetID, operator string) error {
	// validate the operator address
	if _, err := sdk.AccAddressFromBech32(operator); err != nil {
		return xerrors.Errorf(
			"ValidateIDAndOperator: invalid operator address for operator %s", operator,
		)
	}
	_, stakerClientChainID, err := assetstypes.ValidateID(stakerID, true, false)
	if err != nil {
		return xerrors.Errorf(
			"ValidateIDAndOperator: invalid stakerID: %s",
			stakerID,
		)
	}
	_, assetClientChainID, err := assetstypes.ValidateID(assetID, true, false)
	if err != nil {
		return xerrors.Errorf(
			"ValidateIDAndOperator: invalid assetID: %s",
			assetID,
		)
	}
	if stakerClientChainID != assetClientChainID {
		return xerrors.Errorf(
			"ValidateIDAndOperator: the client chain layerZero IDs of the staker and asset are different, stakerID:%s, assetID:%s",
			stakerID, assetID)
	}
	return nil
}

func (gs GenesisState) ValidateAssociations() error {
	// for associations, one stakerID can be associated only with one operator.
	// but one operator may have multiple stakerIDs associated with it.
	associatedStakerIDs := make(map[string]struct{}, len(gs.Associations))
	for _, association := range gs.Associations {
		// check operator address
		if _, err := sdk.AccAddressFromBech32(association.Operator); err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAssociations: invalid operator address for operator %s", association.Operator,
			)
		}
		// check staker address
		if _, _, err := assetstypes.ValidateID(
			association.StakerId, true, true,
		); err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAssociations: invalid staker ID %s: %s", association.StakerId, err,
			)
		}
		// check for duplicate stakerIDs
		if _, ok := associatedStakerIDs[association.StakerId]; ok {
			return ErrInvalidGenesisData.Wrapf("ValidateAssociations: duplicate staker ID %s", association.StakerId)
		}
		associatedStakerIDs[association.StakerId] = struct{}{}
		// we don't check that this `association.stakerID` features in `gs.Delegations`,
		// because we allow the possibility of a staker without any delegations to be associated
		// with an operator.
	}
	return nil
}

func (gs GenesisState) ValidateDelegationStates() error {
	validationFunc := func(_ int, info DelegationStates) error {
		keys, err := ParseStakerAssetIDAndOperator([]byte(info.Key))
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateDelegationStates: %s", err.Error())
		}

		err = ValidateIDAndOperator(keys.StakerId, keys.AssetId, keys.OperatorAddr)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateDelegationStates: %s", err.Error())
		}

		// check that there is no nil value provided.
		if info.States.UndelegatableShare.IsNil() || info.States.PendingUndelegationAmount.IsNil() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateDelegationStates: nil delegation state for %s: %+v",
				info.Key, info,
			)
		}

		// check for negative values.
		if info.States.UndelegatableShare.IsNegative() || info.States.PendingUndelegationAmount.IsNegative() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateDelegationStates: negative delegation state  for %s: %+v",
				info.Key, info,
			)
		}

		return nil
	}
	seenFieldValueFunc := func(info DelegationStates) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.DelegationStates, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateStakerList() error {
	validationFunc := func(_ int, stakersByOperator string) error {
		// validate the key - each Key contains operator + asset id + staker id
		stringList, err := utils.ParseJoinedKeyWithCount([]byte(stakersByOperator), 3)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateStakerList: %s", err.Error())
		}
		// validate the operator address
		if _, err := sdk.AccAddressFromBech32(stringList[0]); err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateStakerList: invalid operator address for operator %s", stringList[0],
			)
		}
		// validate the assetID
		_, assetClientChainID, err := assetstypes.ValidateID(stringList[1], true, false)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateStakerList: invalid assetID: %s",
				stringList[1],
			)
		}
		// validate the staker ID (extracted from the key)
		stakerID := stringList[2]
		_, stakerClientChainID, err := assetstypes.ValidateID(stakerID, true, false)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateStakerList: invalid stakerID: %s",
				stakerID,
			)
		}
		if stakerClientChainID != assetClientChainID {
			return ErrInvalidGenesisData.Wrapf("ValidateStakerList: the client chain layerZero IDs of the staker and asset are different, key:%s stakerID:%s", stakersByOperator, stakerID)
		}
		return nil
	}
	seenFieldValueFunc := func(info string) (string, struct{}) {
		// given an operator o, asset ID a, staker s, the key is o/a/s
		// the following are possible:
		// o1/a1/s1, o1/a1/s2, o1/a2/s1, o1/a2/s2, o2/a1/s1, o2/a1/s1
		// thus, the only invalid form would be repitition of the overall
		// o1/a1/s1 combination.
		return info, struct{}{}
	}
	_, err := utils.CommonValidation(gs.StakersByOperator, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateUndelegations() error {
	validationFunc := func(_ int, undelegationRecord UndelegationAndHoldCount) error {
		undelegation := undelegationRecord.Undelegation
		err := ValidateIDAndOperator(undelegation.StakerId, undelegation.AssetId, undelegation.OperatorAddr)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: %s", err.Error())
		}

		bytes, err := hex.DecodeString(undelegation.TxHash)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: TxHash isn't a hex string, TxHash: %s",
				undelegation.TxHash,
			)
		}
		if len(bytes) != common.HashLength {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: invalid length of TxHash ,TxHash:%s length: %d, should:%d", undelegation.TxHash, len(bytes), common.HashLength)
		}
		if undelegation.ActualCompletedAmount.GT(undelegation.Amount) {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: the completed amount shouldn't be greater than the submitted amount , undelegationRecord：%v", undelegationRecord)
		}
		if undelegation.UndelegationId >= gs.LastUndelegationId {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: the undelegationID should be less than the global undelegationID,undelegationID:%d,globalID:%d", undelegation.UndelegationId, gs.LastUndelegationId)
		}
		if undelegation.CompletedEpochNumber < 0 {
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: negative epoch number in the undelegation: %d",
				undelegation.CompletedEpochNumber,
			)
		}
		switch undelegation.CompletedEpochIdentifier {
		case epochsTypes.NullEpochIdentifier, epochsTypes.MinuteEpochID,
			epochsTypes.HourEpochID, epochsTypes.DayEpochID,
			epochsTypes.WeekEpochID:
		default:
			return ErrInvalidGenesisData.Wrapf("ValidateUndelegations: invalid epoch identifier in the undelegation: %s", undelegation.CompletedEpochIdentifier)
		}
		return nil
	}
	seenFieldValueFunc := func(record UndelegationAndHoldCount) (string, struct{}) {
		return record.Undelegation.TxHash, struct{}{}
	}
	_, err := utils.CommonValidation(gs.Undelegations, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	err := gs.ValidateAssociations()
	if err != nil {
		return err
	}
	err = gs.ValidateDelegationStates()
	if err != nil {
		return err
	}
	err = gs.ValidateStakerList()
	if err != nil {
		return err
	}
	err = gs.ValidateUndelegations()
	if err != nil {
		return err
	}
	err = gs.Params.Validate()
	if err != nil {
		return err
	}
	return nil
}
