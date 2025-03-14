package types

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultDiscountedRateStr default discount rate 10%
	DefaultDiscountedRateStr = "0.10"
	// DefaultPenaltyRateStr default penalty rate 25%
	DefaultPenaltyRateStr = "0.25"
	// DefaultWithdrawalPeriod default withdrawal waiting period
	DefaultWithdrawalPeriod = uint32(30) // 30 epochs
	// DefaultEpochIdentifier default epoch identifier
	DefaultEpochIdentifier = epochstypes.DayEpochID
	// DefaultAmount default amount
	DefaultAmount = 4223372036854775807
)

// DefaultBaseRestakingFee Default base restaking fee
var DefaultBaseRestakingFee = sdk.NewCoin(utils.BaseDenom, sdk.NewInt(DefaultAmount))

func init() {
	// Validate default rates
	if _, err := sdk.NewDecFromStr(DefaultDiscountedRateStr); err != nil {
		panic(fmt.Sprintf("invalid default discounted rate: %v", err))
	}
	if _, err := sdk.NewDecFromStr(DefaultPenaltyRateStr); err != nil {
		panic(fmt.Sprintf("invalid default penalty rate: %v", err))
	}
}

// NewParams creates new parameters instance
func NewParams(
	discountedRate, penaltyRate sdk.Dec,
	baseFee sdk.Coin,
	withdrawalPeriod uint32,
	epochIdentifier string,
) Params {
	return Params{
		DiscountedRate:   discountedRate,
		PenaltyRate:      penaltyRate,
		BaseRestakingFee: &baseFee,
		WithdrawalPeriod: withdrawalPeriod,
		EpochIdentifier:  epochIdentifier,
	}
}

// DefaultParams returns default parameters
func DefaultParams() Params {
	discountedRate := sdk.MustNewDecFromStr(DefaultDiscountedRateStr)
	penaltyRate := sdk.MustNewDecFromStr(DefaultPenaltyRateStr)

	return NewParams(
		discountedRate,
		penaltyRate,
		DefaultBaseRestakingFee,
		DefaultWithdrawalPeriod,
		DefaultEpochIdentifier,
	)
}

// Validate parameter validation
func (p Params) Validate() error {
	if err := ValidateRate(p.DiscountedRate); err != nil {
		return fmt.Errorf("discounted_rate: %w", err)
	}
	if err := ValidateRate(p.PenaltyRate); err != nil {
		return fmt.Errorf("penalty_rate: %w", err)
	}
	if err := ValidateCoin(p.BaseRestakingFee); err != nil {
		return fmt.Errorf("base_restaking_fee: %w", err)
	}
	if p.WithdrawalPeriod == 0 {
		return fmt.Errorf("withdrawal_period cannot be zero")
	}
	return epochstypes.ValidateEpochIdentifierString(p.EpochIdentifier)
}

// ValidateRate validates rate range (0 <= rate <= 1)
func ValidateRate(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNil() {
		return fmt.Errorf("rate cannot be nil")
	}
	if v.IsNegative() || v.GT(sdk.OneDec()) {
		return fmt.Errorf("rate must be in [0, 1] range")
	}
	return nil
}

// ValidateCoin validates coin validity
func ValidateCoin(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !v.IsValid() || v.Amount.IsNegative() {
		return fmt.Errorf("invalid coin: %s", v)
	}
	return nil
}

// String implements string formatting
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// Copy creates parameter copy
func (p Params) Copy() Params {
	return Params{
		DiscountedRate:   p.DiscountedRate,
		PenaltyRate:      p.PenaltyRate,
		BaseRestakingFee: p.BaseRestakingFee,
		WithdrawalPeriod: p.WithdrawalPeriod,
		EpochIdentifier:  p.EpochIdentifier,
	}
}

// OverrideIfRequired override invalid new parameters with previous values
func (p Params) OverrideIfRequired(prev Params, logger log.Logger) Params {
	over := p.Copy()

	// Handle discount rate
	if p.DiscountedRate.IsNil() || p.DiscountedRate.IsNegative() || p.DiscountedRate.GT(sdk.OneDec()) {
		logger.Info("Override discounted_rate", "old", prev.DiscountedRate)
		over.DiscountedRate = prev.DiscountedRate
	}

	// Handle penalty rate
	if p.PenaltyRate.IsNil() || p.PenaltyRate.IsNegative() || p.PenaltyRate.GT(sdk.OneDec()) {
		logger.Info("Override penalty_rate", "old", prev.PenaltyRate)
		over.PenaltyRate = prev.PenaltyRate
	}

	// Handle base fee
	if !p.BaseRestakingFee.IsValid() || p.BaseRestakingFee.Amount.IsNegative() {
		logger.Info("Override base_restaking_fee", "old", prev.BaseRestakingFee)
		over.BaseRestakingFee = prev.BaseRestakingFee
	}

	// Handle withdrawal period
	if p.WithdrawalPeriod == 0 {
		logger.Info("Override withdrawal_period", "old", prev.WithdrawalPeriod)
		over.WithdrawalPeriod = prev.WithdrawalPeriod
	}

	// Handle epoch identifier
	if err := epochstypes.ValidateEpochIdentifierString(p.EpochIdentifier); err != nil {
		logger.Info("Override epoch_identifier", "old", prev.EpochIdentifier)
		over.EpochIdentifier = prev.EpochIdentifier
	}

	return over
}

// Equal parameter equality check
func (p Params) Equal(p2 Params) bool {
	return p.DiscountedRate.Equal(p2.DiscountedRate) &&
		p.PenaltyRate.Equal(p2.PenaltyRate) &&
		p.BaseRestakingFee.IsEqual(*p2.BaseRestakingFee) &&
		p.WithdrawalPeriod == p2.WithdrawalPeriod &&
		p.EpochIdentifier == p2.EpochIdentifier
}
