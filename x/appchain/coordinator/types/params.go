package types

import (
	fmt "fmt"
	time "time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	commontypes "github.com/imua-xyz/imuachain/x/appchain/common/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

const (
	// DefaultTrustingPeriodFraction is the default fraction used to compure the TrustingPeriod as the UnbondingPeriod (of the subcriber chain) * TrustingPeriodFraction.
	DefaultTrustingPeriodFraction = "0.66"

	// DefaultIBCTimeoutPeriod is defined in common/types/shared_params.go
)

var (
	// DefaultInitTimeoutPeriod is the default timeout period for the initial connection handshake. To estimate a time of 1 week, we use a 7-day duration, but add one day for rounding it up. For example, if a message is sent out in the middle of epoch N, it should ideally timeout exactly 7 days after that, which is the middle of epoch N+7. Since we don't track that, we instead use 8 days.
	// A value of 2 weeks was chosen here to ensure that the chain is dropped if it doesn't initialize a connection within a reasonable time frame.
	DefaultInitTimeoutPeriod = epochstypes.NewEpoch(2, "week")

	// DefaultVSCTimeoutPeriod is the default timeout period for the validator set change packet acknowledgment.
	// If a chain goes offline, we will not receive any such packets. To that end, a longer period than the initialization time is chosen for the chain to recover. If it does not recover within this time frame, it will be dropped.
	// The duration is, however, intentionally not kept too permissive.
	DefaultVSCTimeoutPeriod = epochstypes.NewEpoch(4, "week")
)

// DefaultParams returns the default parameters for the module.
func DefaultParams() Params {
	return NewParams(
		ibctmtypes.NewClientState(
			"", // chainID
			ibctmtypes.DefaultTrustLevel,
			0,                    // trusting period
			0,                    // unbonding period
			10*time.Second,       // replaced later so irrelevant
			clienttypes.Height{}, // latest(initial) height
			commitmenttypes.GetSDKSpecs(),
			[]string{"upgrade", "upgradedIBCState"},
		),
		DefaultTrustingPeriodFraction,
		commontypes.DefaultIBCTimeoutPeriod,
		DefaultInitTimeoutPeriod,
		DefaultVSCTimeoutPeriod,
	)
}

// NewParams creates a new Params object
func NewParams(
	cs *ibctmtypes.ClientState,
	trustingPeriodFraction string,
	ibcTimeoutPeriod time.Duration,
	initTimeoutPeriod epochstypes.Epoch,
	vscTimeoutPeriod epochstypes.Epoch,
) Params {
	return Params{
		TemplateClient:         cs,
		TrustingPeriodFraction: trustingPeriodFraction,
		IBCTimeoutPeriod:       ibcTimeoutPeriod,
		InitTimeoutPeriod:      initTimeoutPeriod,
		VSCTimeoutPeriod:       vscTimeoutPeriod,
	}
}

// Validate checks that the parameters have valid values.
func (p Params) Validate() error {
	if p.TemplateClient == nil {
		return fmt.Errorf("template client is nil")
	}
	if err := ValidateTemplateClient(*p.TemplateClient); err != nil {
		return err
	}
	if err := commontypes.ValidateStringFraction(p.TrustingPeriodFraction); err != nil {
		return fmt.Errorf("trusting period fraction is invalid: %s", err)
	}
	if err := commontypes.ValidateDuration(p.IBCTimeoutPeriod); err != nil {
		return fmt.Errorf("IBC timeout period is invalid: %s", err)
	}
	if err := epochstypes.ValidateEpoch(p.InitTimeoutPeriod); err != nil {
		return fmt.Errorf("init timeout period is invalid: %s", err)
	}
	if err := epochstypes.ValidateEpoch(p.VSCTimeoutPeriod); err != nil {
		return fmt.Errorf("VSC timeout period is invalid: %s", err)
	}
	return nil
}

// ValidateTemplateClient validates the client state
func ValidateTemplateClient(i interface{}) error {
	cs, ok := i.(ibctmtypes.ClientState)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T, expected: %T", i, ibctmtypes.ClientState{})
	}

	// copy clientstate to prevent changing original pointer
	copiedClient := cs

	// populate zeroed fields with valid fields
	copiedClient.ChainId = "chainid"

	trustPeriod, err := commontypes.CalculateTrustPeriod(commontypes.DefaultSubscriberUnbondingPeriod, DefaultTrustingPeriodFraction)
	if err != nil {
		return fmt.Errorf("invalid DefaultTrustingPeriodFraction: %T", err)
	}
	copiedClient.TrustingPeriod = trustPeriod

	copiedClient.UnbondingPeriod = commontypes.DefaultSubscriberUnbondingPeriod
	copiedClient.LatestHeight = clienttypes.NewHeight(0, 1)

	if err := copiedClient.Validate(); err != nil {
		return err
	}
	return nil
}
