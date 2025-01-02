package types

import (
	"encoding/hex"

	"github.com/ExocoreNetwork/exocore/utils"

	errorsmod "cosmossdk.io/errors"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/xerrors"
)

// NewGenesis returns a new genesis state with the given inputs.
func NewGenesis(
	associations []StakerToOperator,
	delegationStates []DelegationStates,
	stakersByOperator []StakersByOperator,
	undelegations []UndelegationRecord,
) *GenesisState {
	return &GenesisState{
		Associations:      associations,
		DelegationStates:  delegationStates,
		StakersByOperator: stakersByOperator,
		Undelegations:     undelegations,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return NewGenesis(nil, nil, nil, nil)
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
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"invalid operator address for operator %s", association.Operator,
			)
		}
		// check staker address
		if _, _, err := assetstypes.ValidateID(
			association.StakerId, true, true,
		); err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "invalid staker ID %s: %s", association.StakerId, err,
			)
		}
		// check for duplicate stakerIDs
		if _, ok := associatedStakerIDs[association.StakerId]; ok {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "duplicate staker ID %s", association.StakerId,
			)
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
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}

		err = ValidateIDAndOperator(keys.StakerId, keys.AssetId, keys.OperatorAddr)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}

		// check that there is no nil value provided.
		if info.States.UndelegatableShare.IsNil() || info.States.WaitUndelegationAmount.IsNil() {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"nil delegation state for %s: %+v",
				info.Key, info,
			)
		}

		// check for negative values.
		if info.States.UndelegatableShare.IsNegative() || info.States.WaitUndelegationAmount.IsNegative() {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"negative delegation state  for %s: %+v",
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
		return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}
	return nil
}

func (gs GenesisState) ValidateStakerList() error {
	validationFunc := func(_ int, stakersByOperator StakersByOperator) error {
		// validate the key
		stringList, err := assetstypes.ParseJoinedStoreKey([]byte(stakersByOperator.Key), 2)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}
		// validate the operator address
		if _, err := sdk.AccAddressFromBech32(stringList[0]); err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"invalid operator address for operator %s", stringList[0],
			)
		}
		// validate the assetID
		_, assetClientChainID, err := assetstypes.ValidateID(stringList[1], true, false)
		if err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData,
				"invalid assetID: %s",
				stringList[1],
			)
		}
		// validate the staker list
		stakerValidationFunc := func(_ int, stakerID string) error {
			_, stakerClientChainID, err := assetstypes.ValidateID(stakerID, true, false)
			if err != nil {
				return errorsmod.Wrapf(
					ErrInvalidGenesisData,
					"invalid stakerID: %s",
					stakerID,
				)
			}
			if stakerClientChainID != assetClientChainID {
				return errorsmod.Wrapf(ErrInvalidGenesisData, "the client chain layerZero IDs of the staker and asset are different, key:%s stakerID:%s", stakersByOperator.Key, stakerID)
			}
			return nil
		}
		seenStakerFunc := func(stakerID string) (string, struct{}) {
			return stakerID, struct{}{}
		}
		_, err = utils.CommonValidation(stakersByOperator.Stakers, seenStakerFunc, stakerValidationFunc)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}
		return nil
	}
	seenFieldValueFunc := func(info StakersByOperator) (string, struct{}) {
		return info.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.StakersByOperator, seenFieldValueFunc, validationFunc)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
	}
	return nil
}

func (gs GenesisState) ValidateUndelegations() error {
	validationFunc := func(_ int, undelegation UndelegationRecord) error {
		err := ValidateIDAndOperator(undelegation.StakerId, undelegation.AssetId, undelegation.OperatorAddr)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
		}

		bytes, err := hex.DecodeString(undelegation.TxHash)
		if err != nil {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "TxHash isn't a hex string, TxHash: %s",
				undelegation.TxHash,
			)
		}
		if len(bytes) != common.HashLength {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "invalid length of TxHash ,TxHash:%s length: %d, should:%d",
				undelegation.TxHash, len(bytes), common.HashLength,
			)
		}
		if !undelegation.IsPending {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "all undelegations should be pending, undelegation:%v",
				undelegation,
			)
		}
		if undelegation.CompleteBlockNumber < undelegation.BlockNumber {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "the block number to complete shouldn't be less than the submitted , undelegation：%v",
				undelegation,
			)
		}
		if undelegation.ActualCompletedAmount.GT(undelegation.Amount) {
			return errorsmod.Wrapf(
				ErrInvalidGenesisData, "the completed amount shouldn't be greater than the submitted amount , undelegation：%v",
				undelegation,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(undelegation UndelegationRecord) (string, struct{}) {
		return undelegation.TxHash, struct{}{}
	}
	_, err := utils.CommonValidation(gs.Undelegations, seenFieldValueFunc, validationFunc)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidGenesisData, err.Error())
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
	return nil
}
