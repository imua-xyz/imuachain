package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// interface guard
var _ codec.ProtoMarshaler = &DecValueField{}

// String implements the Stringer interface for DecValueField.
func (d *DecValueField) String() string {
	return d.Amount.String()
}
