package types

import (
	fmt "fmt"
	time "time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultMinCommissionUpdateInterval is the default minimum interval
	// between commission updates. In other words, the operator can only
	// update the commission rate once every 24 hours by default.
	DefaultMinCommissionUpdateInterval = 24 * time.Hour
)

var (
	// DefaultMinCommissionRate is the default minimum commission rate.
	// It is set to 5% by default.
	DefaultMinCommissionRate = sdk.NewDecWithPrec(5, 2)

	// DefaultMaxSlashProportion is the default maximum slash proportion when executing the slash event.
	// It is set to 100% by default.
	DefaultMaxSlashProportion = sdk.OneDec()
)

// NewParams creates a new Params instance.
func NewParams(
	minCommissionUpdateInterval time.Duration,
	minCommissionRate sdk.Dec,
	maxSlashProportion sdk.Dec,
) Params {
	return Params{
		MinCommissionUpdateInterval: minCommissionUpdateInterval,
		MinCommissionRate:           minCommissionRate,
		MaxSlashProportion:          maxSlashProportion,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultMinCommissionUpdateInterval,
		DefaultMinCommissionRate,
		DefaultMaxSlashProportion,
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	// 0 duration is allowed to change commission at will
	if err := ValidateNonNegativeDuration(p.MinCommissionUpdateInterval); err != nil {
		return fmt.Errorf("min commission update interval: %w", err)
	}
	// 0 rate is allowed to permit operators with no commission
	if err := ValidateNonNegativeDec(p.MinCommissionRate); err != nil {
		return fmt.Errorf("min commission rate: %w", err)
	}
	if p.MinCommissionRate.GT(sdk.OneDec()) {
		return fmt.Errorf("min commission rate must be less than or equal to 1")
	}

	if err := ValidateNonNegativeDec(p.MaxSlashProportion); err != nil {
		return fmt.Errorf("max slash proportion: %w", err)
	}
	if p.MaxSlashProportion.GT(sdk.OneDec()) {
		return fmt.Errorf("max slash proportion must be less than or equal to 1")
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

// ValidateNonNegativeDuration checks if the duration is non-negative.
func ValidateNonNegativeDuration(duration time.Duration) error {
	if duration < 0 {
		return fmt.Errorf("duration must be non-negative")
	}
	return nil
}

// ValidateNonNegativeDec checks if the dec is non-negative.
func ValidateNonNegativeDec(dec sdk.Dec) error {
	if dec.IsNil() {
		return fmt.Errorf("dec must be non-nil")
	}
	if dec.IsNegative() {
		return fmt.Errorf("dec must be non-negative")
	}
	return nil
}
