package types

import (
	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewSubscriberValidator creates a new SubscriberValidator instance.
func NewSubscriberValidator(
	address []byte, power int64, pubKey cryptotypes.PubKey,
) (SubscriberValidator, error) {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return SubscriberValidator{}, err
	}

	return SubscriberValidator{
		ConsAddress: address,
		Power:       power,
		Pubkey:      pkAny,
	}, nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces.
// It is required to ensure that ConsPubKey below works.
func (sv SubscriberValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(sv.Pubkey, &pk)
}

// ConsPubKey returns the validator PubKey as a cryptotypes.PubKey.
func (sv SubscriberValidator) ConsPubKey() (cryptotypes.PubKey, error) {
	pk, ok := sv.Pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidType,
			"expecting cryptotypes.PubKey, got %T",
			pk,
		)
	}

	return pk, nil
}
