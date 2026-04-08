package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
	"github.com/stretchr/testify/require"
)

type stubDogfoodHook struct {
	bonded, removed, created error
}

func (s stubDogfoodHook) AfterValidatorBonded(
	_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress,
) error {
	return s.bonded
}

func (s stubDogfoodHook) AfterValidatorRemoved(
	_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress,
) error {
	return s.removed
}

func (s stubDogfoodHook) AfterValidatorCreated(_ sdk.Context, _ sdk.ValAddress) error {
	return s.created
}

type recordRemovedHook struct {
	called *bool
}

func (r recordRemovedHook) AfterValidatorBonded(
	_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress,
) error {
	return nil
}

func (r recordRemovedHook) AfterValidatorRemoved(
	_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress,
) error {
	*r.called = true
	return nil
}

func (r recordRemovedHook) AfterValidatorCreated(_ sdk.Context, _ sdk.ValAddress) error {
	return nil
}

// TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits (checklist E3):
// If a future hook returns an error, MultiDogfoodHooks stops before later hooks. Dogfood
// EndBlock prune then reschedules the consensus addr without writeFunc — no partial prune.
func TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits(t *testing.T) {
	var secondCalled bool
	h := types.NewMultiDogfoodHooks(
		stubDogfoodHook{removed: fmt.Errorf("simulated hook failure")},
		recordRemovedHook{called: &secondCalled},
	)
	err := h.AfterValidatorRemoved(sdk.Context{}, nil, nil)
	require.Error(t, err)
	require.False(t, secondCalled, "later hooks must not run after first error")
}
