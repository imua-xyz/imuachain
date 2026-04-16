package types

// Helper methods for the proto-generated MsgSignCheckpoint.
// The struct itself is defined in tx.pb.go via proto generation.

import (
	"encoding/hex"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

const TypeMsgSignCheckpoint = "sign_checkpoint"

var _ sdk.Msg = &MsgSignCheckpoint{}

func (msg *MsgSignCheckpoint) Route() string { return ModuleName }
func (msg *MsgSignCheckpoint) Type() string  { return TypeMsgSignCheckpoint }

func (msg *MsgSignCheckpoint) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.ValidatorAddress)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSignCheckpoint) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSignCheckpoint) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ValidatorAddress); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, msg.ValidatorAddress)
	}
	if msg.DstChainId == 0 {
		return fmt.Errorf("dst_chain_id must be > 0")
	}
	if msg.CheckpointNonce == 0 {
		return fmt.Errorf("checkpoint_nonce must be > 0")
	}
	if len(msg.ValidatorEvmAddress) != 40 {
		return fmt.Errorf("validator_evm_address must be 40 hex chars")
	}
	if _, err := hex.DecodeString(msg.ValidatorEvmAddress); err != nil {
		return fmt.Errorf("validator_evm_address must be hex: %w", err)
	}
	if msg.V != 27 && msg.V != 28 {
		return fmt.Errorf("v must be 27 or 28, got %d", msg.V)
	}
	if len(msg.R) != 32 {
		return fmt.Errorf("r must be 32 bytes, got %d", len(msg.R))
	}
	if len(msg.S) != 32 {
		return fmt.Errorf("s must be 32 bytes, got %d", len(msg.S))
	}
	return nil
}

// EVMAddress returns the parsed EVM address.
func (msg *MsgSignCheckpoint) EVMAddress() common.Address {
	return common.HexToAddress(msg.ValidatorEvmAddress)
}

// RSBytes returns the R and S values as fixed-size arrays.
func (msg *MsgSignCheckpoint) RSBytes() (r [32]byte, s [32]byte) {
	copy(r[:], msg.R)
	copy(s[:], msg.S)
	return
}
