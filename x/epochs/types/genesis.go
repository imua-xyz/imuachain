package types

import (
	"errors"
	"time"

	errorsmod "cosmossdk.io/errors"
)

func NewGenesisState(epochs []EpochInfo) *GenesisState {
	return &GenesisState{Epochs: epochs}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	epochs := []EpochInfo{
		// alphabetical order
		NewGenesisEpochInfo(DayEpochID, time.Hour*24),
		NewGenesisEpochInfo(HourEpochID, time.Hour),
		NewGenesisEpochInfo(MinuteEpochID, time.Minute),
		NewGenesisEpochInfo(WeekEpochID, time.Hour*24*7),
	}
	return NewGenesisState(epochs)
}

// Validate performs basic stateless genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	epochIdentifiers := map[string]struct{}{}
	for _, epoch := range gs.Epochs {
		if err := epoch.Validate(); err != nil {
			return errorsmod.Wrapf(ErrInvalidGenesisData, "invalid epoch %s", err)
		}
		if _, ok := epochIdentifiers[epoch.Identifier]; ok {
			return errorsmod.Wrap(ErrInvalidGenesisData, "epoch identifier should be unique")
		}
		epochIdentifiers[epoch.Identifier] = struct{}{}
	}
	return nil
}

// Validate validates epoch info. Since it does not particularly pertain to genesis data,
// the error is returned as-is and not wrapped with ErrInvalidGenesisData.
func (epoch EpochInfo) Validate() error {
	if err := ValidateEpochIdentifierString(epoch.Identifier); err != nil {
		return err
	}
	if epoch.Duration <= 0 {
		return errors.New("epoch duration should NOT be non-positive")
	}
	// the EpochNumber may be 0 at genesis so we don't validate that.
	if epoch.CurrentEpoch < 0 {
		return errors.New("epoch CurrentEpoch must be non-negative")
	}
	if epoch.CurrentEpochStartHeight < 0 {
		return errors.New("epoch CurrentEpochStartHeight must be non-negative")
	}
	return nil
}

func NewGenesisEpochInfo(identifier string, duration time.Duration) EpochInfo {
	return EpochInfo{
		Identifier:              identifier,
		StartTime:               time.Time{}, // zero time
		Duration:                duration,
		CurrentEpoch:            0,
		CurrentEpochStartHeight: 0,
		CurrentEpochStartTime:   time.Time{},
		EpochCountingStarted:    false,
	}
}
