package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"gopkg.in/yaml.v2"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
)

const (
	// DefaultEpochsUntilUnbonded is the default number of epochs after which an unbonding entry
	// is released. For example, if an unbonding is requested during epoch 8, it is made
	// effective at the beginning of epoch 9. The unbonding amount is released at the beginning
	// of epoch 16 (9 + DefaultEpochsUntilUnbonded).
	DefaultEpochsUntilUnbonded = 7
	// DefaultEpochIdentifier is the epoch identifier which is used, by default, to identify the
	// epoch. Note that the options in the default genesis include minute, week, hour or day.
	DefaultEpochIdentifier = epochstypes.DayEpochID
	// DefaultMaxValidators is the default maximum number of bonded validators. It is defined as
	// a copy here so that we can use a value other than that in x/staking, if necessary.
	DefaultMaxValidators = stakingtypes.DefaultMaxValidators
	// DefaultHistorical entries is the number of entries of historical staking data to persist.
	// It is defined as a copy here so that we can use a value other than that in x/staking, if
	// necessary.
	DefaultHistoricalEntries = stakingtypes.DefaultHistoricalEntries
)

var (
	// DefaultAssetIDs is the default asset IDs accepted by the dogfood module.
	DefaultAssetIDs = []string{}
	// DefaultMinSelfDelegation is the default minimum self-delegation amount for a validator.
	// It is denominated in USD. We do not support cents, since it is an integer.
	DefaultMinSelfDelegation = sdkmath.ZeroInt()
)

// NewParams creates a new Params instance.
func NewParams(
	epochsUntilUnbonded uint32,
	epochIdentifier string,
	maxValidators uint32,
	historicalEntries uint32,
	assetIDs []string,
	minSelfDelegation sdkmath.Int,
) Params {
	return Params{
		EpochsUntilUnbonded: epochsUntilUnbonded,
		EpochIdentifier:     epochIdentifier,
		MaxValidators:       maxValidators,
		HistoricalEntries:   historicalEntries,
		AssetIDs:            assetIDs,
		MinSelfDelegation:   minSelfDelegation,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultEpochsUntilUnbonded,
		DefaultEpochIdentifier,
		DefaultMaxValidators,
		DefaultHistoricalEntries,
		DefaultAssetIDs,
		DefaultMinSelfDelegation,
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := ValidatePositiveUint32(p.EpochsUntilUnbonded); err != nil {
		return fmt.Errorf("epochs until unbonded: %w", err)
	}
	if err := epochstypes.ValidateEpochIdentifierInterface(p.EpochIdentifier); err != nil {
		return fmt.Errorf("epoch identifier: %w", err)
	}
	if err := ValidatePositiveUint32(p.MaxValidators); err != nil {
		return fmt.Errorf("max validators: %w", err)
	}
	if err := ValidatePositiveUint32(p.HistoricalEntries); err != nil {
		return fmt.Errorf("historical entries: %w", err)
	}
	if err := ValidateAssetIDs(p.AssetIDs); err != nil {
		return fmt.Errorf("asset IDs: %w", err)
	}
	if err := ValidateNonNegativeInt(p.MinSelfDelegation); err != nil {
		return fmt.Errorf("min self delegation: %w", err)
	}
	return nil
}

// ValidatePositiveUint32 checks whether the supplied value is a positive uint32.
func ValidatePositiveUint32(i interface{}) error {
	if val, ok := i.(uint32); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if val == 0 {
		return fmt.Errorf("invalid parameter value: %d", val)
	}
	return nil
}

// String implements the Stringer interface. Ths interface is required as part of the
// proto.Message interface, which is used in the query server.
func (p Params) String() string {
	out, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}
	return string(out)
}

// ValidateAssetIDs checks whether the supplied value is a valid asset ID.
func ValidateAssetIDs(i interface{}) error {
	var val []string
	var ok bool
	if val, ok = i.([]string); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if len(val) == 0 {
		return fmt.Errorf("invalid parameter value: %v", val)
	}
	// check for duplicates
	seen := make(map[string]struct{})
	for _, assetID := range val {
		if _, _, err := assetstypes.ParseID(assetID); err != nil {
			return fmt.Errorf("invalid parameter value: %v", val)
		}
		if _, ok := seen[assetID]; ok {
			return fmt.Errorf("duplicate asset ID: %s", assetID)
		}
		seen[assetID] = struct{}{}
	}
	return nil
}

// ValidateNonNegativeInt checks whether the supplied value is a non-negative integer.
func ValidateNonNegativeInt(i interface{}) error {
	if val, ok := i.(sdkmath.Int); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if val.IsNil() {
		return fmt.Errorf("nil parameter value: %s", val)
	} else if val.IsNegative() {
		return fmt.Errorf("invalid parameter value: %s", val)
	}
	return nil
}
