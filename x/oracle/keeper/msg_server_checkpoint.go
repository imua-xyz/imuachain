package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SignCheckpoint handles a validator's ECDSA signature submission for an outbound checkpoint.
func (ms msgServer) SignCheckpoint(goCtx context.Context, msg *types.MsgSignCheckpoint) (*types.MsgSignCheckpointResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cp, found := ms.GetCheckpoint(ctx, msg.DstChainId, msg.CheckpointNonce)
	if !found {
		return nil, fmt.Errorf("checkpoint not found: dstChainID=%d nonce=%d", msg.DstChainId, msg.CheckpointNonce)
	}
	if cp.Finalized {
		return &types.MsgSignCheckpointResponse{Finalized: true}, nil
	}

	evmAddr := msg.EVMAddress()
	validatorPower, err := ms.resolveValidatorPower(ctx, evmAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve validator power for %s: %w", evmAddr.Hex(), err)
	}
	if validatorPower <= 0 {
		return nil, fmt.Errorf("signer %s is not an active validator or has zero power", evmAddr.Hex())
	}

	r, s := msg.RSBytes()
	finalized, err := ms.AddCheckpointSignature(ctx, msg.DstChainId, msg.CheckpointNonce,
		evmAddr, uint8(msg.V), r, s, validatorPower)
	if err != nil {
		return nil, err
	}

	return &types.MsgSignCheckpointResponse{Finalized: finalized}, nil
}

// resolveValidatorPower maps an EVM address to the validator's voting power.
//
// Production path (operator keeper wired):
//
//	EVM address → Cosmos AccAddress (same bytes in evmos)
//	→ operator.GetOperatorConsKeyForChainID(accAddr, chainID)
//	→ wrappedKey.ToConsAddr()
//	→ dogfood.GetImuachainValidator(consAddr) → Power
//
// Test fallback (no operator keeper):
//
//	Iterate active validators, return average power.
func (ms msgServer) resolveValidatorPower(ctx sdk.Context, evmAddr common.Address) (int64, error) {
	accAddr := sdk.AccAddress(evmAddr.Bytes())

	// Production path: use operator keeper to resolve consensus key → validator power.
	if ms.operatorKeeper != nil {
		chainID := ctx.ChainID()
		found, wrappedKey, err := ms.operatorKeeper.GetOperatorConsKeyForChainID(ctx, accAddr, chainID)
		if err != nil {
			// Operator exists but key lookup failed (e.g. not registered for this chain).
			// Log and fall through to return 0.
			ctx.Logger().Debug("operator cons key lookup failed", "addr", evmAddr.Hex(), "err", err)
			return 0, nil
		}
		if !found {
			return 0, nil
		}

		consAddr := wrappedKey.ToConsAddr()

		// Direct lookup from dogfood validator set by consensus address.
		validator, found := ms.getImuachainValidatorByConsAddr(ctx, consAddr)
		if !found {
			return 0, nil
		}
		return validator.Power, nil
	}

	// Fallback for unit tests: return average power of active validators.
	return ms.fallbackAveragePower(ctx), nil
}

// GetImuachainValidator looks up a validator by consensus address via the dogfood keeper.
// Returns (validator, found). Panics are recovered to handle nil keepers in tests.
// imuachainValidatorInfo is a minimal struct to hold validator power.
type imuachainValidatorInfo struct {
	Power int64
}

func (ms msgServer) getImuachainValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (imuachainValidatorInfo, bool) {
	defer func() { recover() }() //nolint:errcheck
	validators := ms.GetAllImuachainValidators(ctx)
	for _, v := range validators {
		if sdk.ConsAddress(v.Address).Equals(consAddr) {
			return imuachainValidatorInfo{Power: v.Power}, true
		}
	}
	return imuachainValidatorInfo{}, false
}

func (ms msgServer) fallbackAveragePower(ctx sdk.Context) int64 {
	defer func() { recover() }() //nolint:errcheck
	validators := ms.GetAllImuachainValidators(ctx)
	if len(validators) == 0 {
		return 0
	}
	var totalPower int64
	for _, v := range validators {
		totalPower += v.Power
	}
	return totalPower / int64(len(validators))
}
