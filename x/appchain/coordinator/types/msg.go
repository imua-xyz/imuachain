package types

import (
	fmt "fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

const TypeRegisterSubscriberChain = "register_subscriber_chain"

var _ sdk.Msg = &RegisterSubscriberChainRequest{}

func (msg *RegisterSubscriberChainRequest) Route() string {
	return RouterKey
}

func (msg *RegisterSubscriberChainRequest) Type() string {
	return TypeRegisterSubscriberChain
}

// GetSignBytes returns the raw bytes for a RegisterSubscriberChainRequest message that
// the expected signer needs to sign.
func (msg *RegisterSubscriberChainRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic executes sanity validation on the provided data
func (msg *RegisterSubscriberChainRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return sdkerrors.Wrap(err, "invalid from address")
	}
	// the chainID must be non-empty; there is no maximum length imposed
	chainID := strings.TrimSpace(msg.ChainID)
	if val := len(chainID); val == 0 {
		return fmt.Errorf("invalid chain id %s %d", chainID, val)
	}
	if epochstypes.ValidateEpochIdentifierString(msg.EpochIdentifier) != nil {
		return fmt.Errorf("invalid epoch identifier %s", msg.EpochIdentifier)
	}
	for _, assetID := range msg.AssetIDs {
		if _, _, err := assetstypes.ValidateID(
			assetID,
			true, // the caller must make them lowercase
			true, // TODO: we support only Ethereum assets for now.
		); err != nil {
			return fmt.Errorf("invalid asset id %s: %s", assetID, err)
		}
	}
	if msg.MaxValidators == 0 {
		return fmt.Errorf("invalid max validators %d", msg.MaxValidators)
	}
	// no need to validate min self delegation, it is a uint64
	if err := msg.SubscriberParams.Validate(); err != nil {
		return fmt.Errorf("invalid subscriber params: %s", err)
	}
	return nil
}

// GetSigners returns the expected signers for a RegisterSubscriberChainRequest message
func (msg *RegisterSubscriberChainRequest) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		// same behavior as cosmos-sdk
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// NewRegisterSubscriberChainRequest creates a new RegisterSubscriberChainRequest using
// the provided creator and json value. If the JSON is not valid, an error is returned.
func NewRegisterSubscriberChainRequest(
	creator, jsonValue string,
) (*RegisterSubscriberChainRequest, error) {
	r := &RegisterSubscriberChainRequest{}
	if err := json.Unmarshal([]byte(jsonValue), r); err != nil {
		return nil, err
	}
	// the creator is overwritten
	r.FromAddress = creator
	return r, nil
}
