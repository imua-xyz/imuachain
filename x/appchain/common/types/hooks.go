package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// interface guard
var _ SubscriberHooks = &MultiSubscriberHooks{}

// MultiSubscriberHooks is a collection of SubscriberHooks. It calls the hook for each element in the
// collection one-by-one. The hook is called in the order in which the collection is created.
type MultiSubscriberHooks []SubscriberHooks

// NewMultiSubscriberHooks is used to create a collective object of subscriber hooks from a list of
// the hooks. It follows the "accept interface, return concrete types" philosophy. Other modules
// may set the hooks by calling k := (*k).SetHooks(NewMultiSubscriberHooks(hookI,hookJ))
func NewMultiSubscriberHooks(hooks ...SubscriberHooks) MultiSubscriberHooks {
	return hooks
}

// AfterValidatorBonded is the implementation of types.SubscriberHooks for MultiSubscriberHooks.
func (hooks MultiSubscriberHooks) AfterValidatorBonded(
	ctx sdk.Context,
	consAddr sdk.ConsAddress,
	operator sdk.ValAddress,
) error {
	for _, hook := range hooks {
		if err := hook.AfterValidatorBonded(ctx, consAddr, operator); err != nil {
			return err
		}
	}
	return nil
}
