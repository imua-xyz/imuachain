package types

import (
	"fmt"
	"strings"
)

const (
	// DayEpochID defines the identifier for a daily epoch.
	DayEpochID = "day"
	// HourEpochID defines the identifier for an hourly epoch.
	HourEpochID = "hour"
	// MinuteEpochID defines the identifier for an epoch that is a minute long.
	MinuteEpochID = "minute"
	// WeekEpochID defines the identifier for a weekly epoch.
	WeekEpochID = "week"

	// NullEpochIdentifier and NullEpochNumber are used to construct the key for undelegations
	// that aren't restricted by any AVS unbonding configuration.
	// Since it's a virtual epoch configuration, it shouldn't be configured in the genesis.
	NullEpochIdentifier = "NullEpoch"
	NullEpochNumber     = int64(0)
)

// ValidateEpochIdentifierInterface accepts an interface and validates it as an epoch
// identifier. It is not used directly by this module; rather it is created for other
// modules to validate their params.
func ValidateEpochIdentifierInterface(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := ValidateEpochIdentifierString(v); err != nil {
		return err
	}

	return nil
}

// ValidateEpochIdentifierString accepts a string and validates it as an epoch identifier. It
// is a convenience method more often used by other modules.
func ValidateEpochIdentifierString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("empty epoch identifier: %+v", s)
	}
	if s == NullEpochIdentifier {
		return fmt.Errorf("epoch identifier cannot be the null epoch identifier: %+v", s)
	}
	return nil
}

// ValidateEpoch accepts an Epoch and validates it. The validation performed is that the epoch identifier string is
// valid, and that the epoch number (uint64) is not zero.
func ValidateEpoch(epoch Epoch) error {
	if err := ValidateEpochIdentifierString(epoch.EpochIdentifier); err != nil {
		return err
	}
	if epoch.EpochNumber == 0 {
		return fmt.Errorf("epoch number cannot be zero")
	}
	return nil
}

// ValidateEpochInterface accepts an interface and validates it as an Epoch.
func ValidateEpochInterface(i interface{}) error {
	v, ok := i.(Epoch)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return ValidateEpoch(v)
}
