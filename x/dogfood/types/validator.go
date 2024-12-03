package types

import (
	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewExocoreValidator creates an ExocoreValidator with the specified (consensus) address, vote
// power and consensus public key. If the public key is malformed, it returns an error.
func NewExocoreValidator(
	address []byte, power int64, pubKey cryptotypes.PubKey,
) (ExocoreValidator, error) {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return ExocoreValidator{}, errorsmod.Wrap(err, "failed to pack public key")
	}
	return ExocoreValidator{
		Address: address,
		Power:   power,
		Pubkey:  pkAny,
	}, nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces.
// It is required to ensure that ConsPubKey below works.
func (ecv ExocoreValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pk cryptotypes.PubKey
	return unpacker.UnpackAny(ecv.Pubkey, &pk)
}

// ConsPubKey returns the validator PubKey as a cryptotypes.PubKey.
func (ecv ExocoreValidator) ConsPubKey() (cryptotypes.PubKey, error) {
	pk, ok := ecv.Pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, sdkerrors.ErrInvalidType.Wrap("fail to get the expected type cryptotypes.PubKey")
	}

	return pk, nil
}
