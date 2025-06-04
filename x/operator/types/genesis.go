package types

import (
	"strings"

	epochtypes "github.com/imua-xyz/imuachain/x/epochs/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

func NewGenesisState(
	operators []OperatorDetail,
	operatorConsKeys []OperatorConsKeyRecord,
	optStates []OptedState,
	operatorUSDValues []OperatorUSDValue,
	avsUSDValues []AVSUSDValue,
	slashStates []OperatorSlashState,
	prevConsKeys []PrevConsKey,
	operatorKeyRemovals []OperatorKeyRemoval,
	operatorAssetUSDValues []OperatorAssetUSDValue,
) *GenesisState {
	return &GenesisState{
		Operators:              operators,
		OperatorRecords:        operatorConsKeys,
		OptStates:              optStates,
		OperatorUSDValues:      operatorUSDValues,
		AVSUSDValues:           avsUSDValues,
		SlashStates:            slashStates,
		PreConsKeys:            prevConsKeys,
		OperatorKeyRemovals:    operatorKeyRemovals,
		OperatorAssetUsdValues: operatorAssetUSDValues,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return NewGenesisState(nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

// ValidateOperators rationale for the validation:
//  1. since this function should support chain restarts and upgrades, we cannot require
//     the format of the earnings address be EVM only.
func (gs GenesisState) ValidateOperators() (map[string]struct{}, error) {
	// checks list:
	// - no duplicate addresses in `gs.Operators`.
	// - correct bech32 format for each address in `gs.Operators`
	// - no `chainID` duplicates for earnings addresses list in `gs.Operators`.
	operators := make(map[string]struct{}, len(gs.Operators))
	for _, op := range gs.Operators {
		address := op.OperatorAddress
		if _, found := operators[address]; found {
			return nil, ErrInvalidGenesisData.Wrapf(
				"ValidateOperators: duplicate operator address %s", address,
			)
		}
		_, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			return nil, ErrInvalidGenesisData.Wrapf(
				"ValidateOperators: invalid operator address %s: %s", address, err,
			)
		}
		if op.OperatorInfo.EarningsAddr != address {
			return nil, ErrInvalidGenesisData.Wrapf(
				"operator address %s has earnings address %s", address, op.OperatorInfo.EarningsAddr,
			)
		}
		if op.OperatorInfo.ApproveAddr != address {
			return nil, ErrInvalidGenesisData.Wrapf(
				"operator address %s has approve address %s", address, op.OperatorInfo.ApproveAddr,
			)
		}
		operators[address] = struct{}{}
		if op.OperatorInfo.ClientChainEarningsAddr != nil {
			lzIDs := make(map[uint64]struct{}, len(op.OperatorInfo.ClientChainEarningsAddr.EarningInfoList))
			for _, info := range op.OperatorInfo.ClientChainEarningsAddr.EarningInfoList {
				lzID := info.LzClientChainID
				if _, found := lzIDs[lzID]; found {
					return nil, ErrInvalidGenesisData.Wrapf(
						"ValidateOperators: duplicate lz client chain id %d", lzID,
					)
				}
				lzIDs[lzID] = struct{}{}
				// TODO: when moving to support non-EVM chains, this check should be modified
				// to work based on the `lzID` or possibly removed.
				if !common.IsHexAddress(info.ClientChainEarningAddr) {
					return nil, ErrInvalidGenesisData.Wrapf(
						"ValidateOperators: invalid client chain earning address %s", info.ClientChainEarningAddr,
					)
				}
			}
		}
		if op.OperatorInfo.Commission.CommissionRates.Rate.IsNil() ||
			op.OperatorInfo.Commission.CommissionRates.MaxRate.IsNil() ||
			op.OperatorInfo.Commission.CommissionRates.MaxChangeRate.IsNil() {
			return nil, ErrInvalidGenesisData.Wrapf(
				"ValidateOperators: missing commission for operator %s", address,
			)
		}
		if err := op.OperatorInfo.Commission.Validate(); err != nil {
			return nil, ErrInvalidGenesisData.Wrapf(
				"ValidateOperators: invalid commission for operator %s: %s", address, err,
			)
		}
	}
	return operators, nil
}

// ValidateOperatorConsKeyRecords rationale for the validation:
//  2. since the operator module is not meant to handle dogfooding, we should not check
//     whether an operator has keys defined for our chainID. this is left for the dogfood
//     module.
func (gs GenesisState) ValidateOperatorConsKeyRecords(operators map[string]struct{}) error {
	// - correct bech32 format for each address in `gs.OperatorRecords`.
	// - no duplicate addresses in `gs.OperatorRecords`.
	// - no operator that is in `gs.OperatorRecords` but not in `gs.Operators`.
	// - validity of consensus key format for each entry in `gs.OperatorRecords`.
	// - within each chainID, no duplicate consensus keys.
	operatorRecords := make(map[string]struct{}, len(gs.OperatorRecords))
	keysByChainID := make(map[string]map[string]struct{})
	for _, record := range gs.OperatorRecords {
		addr := record.OperatorAddress
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorConsKeyRecords: invalid operator address %s: %s", record.OperatorAddress, err,
			)
		}
		if _, found := operatorRecords[addr]; found {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorConsKeyRecords: duplicate operator record for operator %s", addr,
			)
		}
		operatorRecords[addr] = struct{}{}
		if _, opFound := operators[addr]; !opFound {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorConsKeyRecords: operator record for un-registered operator %s", addr,
			)
		}
		for _, chain := range record.Chains {
			chainID := chain.ChainID
			if !utils.IsValidChainIDWithoutRevision(chainID) {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateOperatorConsKeyRecords: invalid chainID without revision, operator %s: chainID: %s", addr, chainID,
				)
			}
			// Cosmos does not describe a specific `chainID` format, so can't validate it.
			if _, found := keysByChainID[chainID]; !found {
				keysByChainID[chainID] = make(map[string]struct{})
			}

			if wrappedKey := keytypes.NewWrappedConsKeyFromHex(
				chain.ConsensusKey,
			); wrappedKey == nil {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateOperatorConsKeyRecords: invalid consensus key for operator %s: %s", addr, chain.ConsensusKey,
				)
			}

			// within a chain id, there should not be duplicate consensus keys
			if _, found := keysByChainID[chainID][chain.ConsensusKey]; found {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateOperatorConsKeyRecords: duplicate consensus key for operator %s on chain %s", addr, chainID,
				)
			}
			keysByChainID[chainID][chain.ConsensusKey] = struct{}{}
		}
	}
	return nil
}

func (gs GenesisState) ValidateOptedStates(operators map[string]struct{}) (map[string]struct{}, error) {
	avs := make(map[string]struct{})
	validationFunc := func(_ int, state OptedState) error {
		stringList, err := assetstypes.ParseJoinedStoreKey([]byte(state.Key), 2)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOptedStates: can't parse the joined key: %s", err.Error())
		}
		operator, avsAddr := stringList[0], stringList[1]
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOptedStates: unknown operator address for the opted state, %+v",
				state,
			)
		}
		if state.OptInfo.OptedOutHeight < state.OptInfo.OptedInHeight {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOptedStates: the opted-out height should be greater than the opted-in height, %+v",
				state,
			)
		}
		if strings.ToLower(avsAddr) != avsAddr {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOptedStates: contains uppercase characters for avs address: %s", avsAddr,
			)
		}
		if !common.IsHexAddress(avsAddr) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOptedStates: the AVS address isn't an ethereum hex address, %+v",
				state,
			)
		}
		// check whether the number of jail heights matches the status.
		if (state.OptInfo.Jailed && len(state.OptInfo.JailToggleHeights)%2 != 1) ||
			(!state.OptInfo.Jailed && len(state.OptInfo.JailToggleHeights)%2 != 0) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOptedStates: the number of jail heights doesn't match the status, %+v",
				state,
			)
		}
		for i := 1; i < len(state.OptInfo.JailToggleHeights); i++ {
			if state.OptInfo.JailToggleHeights[i] <= state.OptInfo.JailToggleHeights[i-1] {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateOptedStates: invalid jail toggle heights, %+v",
					state,
				)
			}
		}
		avs[avsAddr] = struct{}{}
		return nil
	}
	seenFieldValueFunc := func(state OptedState) (string, struct{}) {
		return state.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.OptStates, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, err
	}
	return avs, nil
}

func (gs GenesisState) ValidateAVSUSDValues(optedAVS map[string]struct{}) (map[string]DecValueField, error) {
	avsUSDValueMap := make(map[string]DecValueField, 0)
	validationFunc := func(_ int, avsUSDValue AVSUSDValue) error {
		if !common.IsHexAddress(avsUSDValue.AVSAddr) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAVSUSDValues: the AVS address isn't an ethereum hex address, avsUSDValue: %+v",
				avsUSDValue,
			)
		}
		if _, ok := optedAVS[avsUSDValue.AVSAddr]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAVSUSDValues: the avs address should be in the opted-in map, avsUSDValue: %+v", avsUSDValue,
			)
		}
		if avsUSDValue.Value.Amount.IsNil() ||
			avsUSDValue.Value.Amount.IsNegative() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateAVSUSDValues: avsUSDValue is nil or negative, avsUSDValue: %+v", avsUSDValue,
			)
		}
		avsUSDValueMap[avsUSDValue.AVSAddr] = avsUSDValue.Value
		return nil
	}
	seenFieldValueFunc := func(usdValue AVSUSDValue) (string, struct{}) {
		return usdValue.AVSAddr, struct{}{}
	}
	_, err := utils.CommonValidation(gs.AVSUSDValues, seenFieldValueFunc, validationFunc)
	if err != nil {
		return nil, err
	}
	return avsUSDValueMap, nil
}

func (gs GenesisState) ValidateOperatorUSDValues(operators map[string]struct{}, avsUSDValues map[string]DecValueField) error {
	validationFunc := func(_ int, operatorUSDValue OperatorUSDValue) error {
		if operatorUSDValue.OptedUSDValue.SelfUSDValue.IsNil() ||
			operatorUSDValue.OptedUSDValue.TotalUSDValue.IsNil() ||
			operatorUSDValue.OptedUSDValue.ActiveUSDValue.IsNil() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: nil field in operatorUSDValue: %+v",
				operatorUSDValue,
			)
		}
		if operatorUSDValue.OptedUSDValue.SelfUSDValue.IsNegative() ||
			operatorUSDValue.OptedUSDValue.TotalUSDValue.IsNegative() ||
			operatorUSDValue.OptedUSDValue.ActiveUSDValue.IsNegative() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: negative field in operatorUSDValue: %+v",
				operatorUSDValue,
			)
		}
		stringList, err := assetstypes.ParseJoinedStoreKey([]byte(operatorUSDValue.Key), 2)
		if err != nil {
			return ErrInvalidGenesisData.Wrap(err.Error())
		}
		avsAddress, operator := stringList[0], stringList[1]
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: unknown operator address for the voting power, %+v",
				operatorUSDValue,
			)
		}
		avsUSDValue, ok := avsUSDValues[avsAddress]
		if !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: the parsed AVS address should be in the avsUSDValues map, AVS: %s, avsUSDValues: %+v",
				avsAddress, avsUSDValues,
			)
		}

		if operatorUSDValue.OptedUSDValue.TotalUSDValue.GT(avsUSDValue.Amount) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: the total USD value of operator shouldn't be greater than the total USD value of the AVS, avsUSDValue: %s, operatorUSDValue: %+v",
				avsUSDValue.Amount.String(), operatorUSDValue,
			)
		}

		if operatorUSDValue.OptedUSDValue.SelfUSDValue.GT(operatorUSDValue.OptedUSDValue.TotalUSDValue) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: the operator's self USD value shouldn't be greater than its total USD value, operatorUSDValue: %+v", operatorUSDValue,
			)
		}

		if operatorUSDValue.OptedUSDValue.ActiveUSDValue.GT(operatorUSDValue.OptedUSDValue.TotalUSDValue) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorUSDValues: the operator's active USD value shouldn't be greater than its total USD value, operatorUSDValue: %+v", operatorUSDValue,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(vp OperatorUSDValue) (string, struct{}) {
		return vp.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.OperatorUSDValues, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateSlashStates(operators, avs map[string]struct{}) error {
	validationFunc := func(_ int, slash OperatorSlashState) error {
		stringList, err := assetstypes.ParseJoinedStoreKey([]byte(slash.Key), 3)
		if err != nil {
			return ErrInvalidGenesisData.Wrap(err.Error())
		}
		operator, avsAddr := stringList[0], stringList[1]
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: unknown operator address for the slashing state, %+v",
				slash,
			)
		}
		// check whether the AVS is in the opted states.
		// This check might be removed if the opted-in states are deleted when
		// the operator opts out of the AVS.
		if _, ok := avs[avsAddr]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: unknown AVS address for the slashing state, %+v",
				slash,
			)
		}
		if slash.Info.EventHeight > slash.Info.SubmittedHeight {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: the submitted height shouldn't be greater than the event height for a slashing record, %+v",
				slash,
			)
		}
		if slash.Info.SlashProportion.IsNil() || slash.Info.SlashProportion.LTE(sdkmath.LegacyZeroDec()) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: invalid slash proportion, it's nil, zero, or negative: %+v",
				slash,
			)
		}

		// validate the slashing execution information
		// the actual executed proportion and value might be zero because of the rounding in an extreme case
		if slash.Info.ExecutionInfo.SlashProportion.IsNil() || slash.Info.ExecutionInfo.SlashProportion.IsNegative() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: invalid slashing execution proportion, it's nil, or negative: %+v",
				slash,
			)
		}
		if slash.Info.ExecutionInfo.SlashValue.IsNil() || slash.Info.ExecutionInfo.SlashValue.IsNegative() {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateSlashStates: invalid slashing execution value, it's nil, or negative: %+v",
				slash,
			)
		}
		// validate the slashing record regarding undelegation
		SlashFromUndelegationVal := func(_ int, slashFromUndelegation SlashFromUndelegation) error {
			if slashFromUndelegation.Amount.IsNil() || slashFromUndelegation.Amount.LTE(sdkmath.ZeroInt()) {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateSlashStates: invalid slashing amount from the undelegation, it's nil, zero, or negative: %+v",
					slash,
				)
			}
			return nil
		}
		seenFieldValueFunc := func(slashFromUndelegation SlashFromUndelegation) (string, struct{}) {
			key := assetstypes.GetJoinedStoreKey(slashFromUndelegation.StakerID, slashFromUndelegation.AssetID)
			return string(key), struct{}{}
		}
		_, err = utils.CommonValidation(slash.Info.ExecutionInfo.SlashUndelegations, seenFieldValueFunc, SlashFromUndelegationVal)
		if err != nil {
			return err
		}
		// validate the slashing record regarding assets pool
		SlashFromAssetsPoolVal := func(_ int, slashFromAssetsPool SlashFromAssetsPool) error {
			// when the data is exported, no check for 0 value is added, that is, even 0 values are exported.
			// to maintain consistency, we allow 0 values here.
			if slashFromAssetsPool.Amount.IsNil() || slashFromAssetsPool.Amount.LT(sdkmath.ZeroInt()) {
				return ErrInvalidGenesisData.Wrapf(
					"ValidateSlashStates: invalid slashing amount from the assets pool, it's nil or negative: %+v",
					slash,
				)
			}
			return nil
		}
		SlashFromAssetsPooLSeenFunc := func(slashFromAssetsPool SlashFromAssetsPool) (string, struct{}) {
			return slashFromAssetsPool.AssetID, struct{}{}
		}
		_, err = utils.CommonValidation(slash.Info.ExecutionInfo.SlashAssetsPool, SlashFromAssetsPooLSeenFunc, SlashFromAssetsPoolVal)
		if err != nil {
			return err
		}
		return nil
	}
	seenFieldValueFunc := func(slash OperatorSlashState) (string, struct{}) {
		return slash.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.SlashStates, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidatePrevConsKeys(operators map[string]struct{}) error {
	validationFunc := func(_ int, prevConsKey PrevConsKey) error {
		keys, err := assetstypes.ParseJoinedStoreKey([]byte(prevConsKey.Key), 2)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidatePrevConsKeys: ValidatePrevConsKeys can't parse the combined key, %+v",
				prevConsKey,
			)
		}

		chainID, operator := keys[0], keys[1]
		if !utils.IsValidChainIDWithoutRevision(chainID) {
			return ErrInvalidGenesisData.Wrapf(
				"ValidatePrevConsKeys: invalid chainID without revision, operator %s: chainID: %s", operator, chainID,
			)
		}
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidatePrevConsKeys: unknown operator address for the previous consensus key, %+v",
				prevConsKey,
			)
		}
		if wrappedKey := keytypes.NewWrappedConsKeyFromHex(
			prevConsKey.ConsensusKey,
		); wrappedKey == nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidatePrevConsKeys: invalid previous consensus key for operator, %+v", prevConsKey,
			)
		}
		// todo: not sure if the duplication of previous consensus keys needs to be checked
		return nil
	}
	seenFieldValueFunc := func(prevConsKey PrevConsKey) (string, struct{}) {
		return prevConsKey.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.PreConsKeys, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorKeyRemovals(operators map[string]struct{}) error {
	validationFunc := func(_ int, operatorKeyRemoval OperatorKeyRemoval) error {
		keys, err := assetstypes.ParseJoinedStoreKey([]byte(operatorKeyRemoval.Key), 2)
		if err != nil {
			return err
		}
		operator := keys[0]
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorKeyRemovals: unknown operator address for the operator key removal, %+v",
				operatorKeyRemoval,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(operatorKeyRemoval OperatorKeyRemoval) (string, struct{}) {
		return operatorKeyRemoval.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.OperatorKeyRemovals, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

func (gs GenesisState) ValidateOperatorAssetUSDValues(operators map[string]struct{}) error {
	if len(gs.OperatorUSDValues) != 0 && len(gs.OperatorAssetUsdValues) == 0 {
		return ErrInvalidGenesisData.Wrap("ValidateOperatorAssetUSDValues: the USD value of the operator's asset can't be empty.")
	}
	validationFunc := func(_ int, usdValue OperatorAssetUSDValue) error {
		stringList, err := assetstypes.ParseJoinedStoreKey([]byte(usdValue.Key), 3)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorAssetUSDValues: can't parse the joined key: %s, err:%s", usdValue.Key, err.Error())
		}
		epochIdentifier, operator, assetID := stringList[0], stringList[1], stringList[2]
		err = epochtypes.ValidateEpochIdentifierString(epochIdentifier)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf("ValidateOperatorAssetUSDValues: invalid epoch identifier,key: %s, err:%s", usdValue.Key, err.Error())
		}
		// check that the operator is registered
		if _, ok := operators[operator]; !ok {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateOperatorAssetUSDValues: unknown operator address for the opted usdValue, %+v",
				usdValue,
			)
		}
		_, _, err = assetstypes.ValidateID(assetID, true, false)
		if err != nil {
			return ErrInvalidGenesisData.Wrapf(
				"ValidateDeposits: invalid assetID: %s",
				assetID,
			)
		}
		return nil
	}
	seenFieldValueFunc := func(usdValue OperatorAssetUSDValue) (string, struct{}) {
		return usdValue.Key, struct{}{}
	}
	_, err := utils.CommonValidation(gs.OperatorAssetUsdValues, seenFieldValueFunc, validationFunc)
	if err != nil {
		return err
	}
	return nil
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	operators, err := gs.ValidateOperators()
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorConsKeyRecords(operators)
	if err != nil {
		return err
	}
	avsMap, err := gs.ValidateOptedStates(operators)
	if err != nil {
		return err
	}
	avsUSDValueMap, err := gs.ValidateAVSUSDValues(avsMap)
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorUSDValues(operators, avsUSDValueMap)
	if err != nil {
		return err
	}
	err = gs.ValidateSlashStates(operators, avsMap)
	if err != nil {
		return err
	}
	err = gs.ValidatePrevConsKeys(operators)
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorKeyRemovals(operators)
	if err != nil {
		return err
	}
	err = gs.ValidateOperatorAssetUSDValues(operators)
	if err != nil {
		return err
	}
	return nil
}
