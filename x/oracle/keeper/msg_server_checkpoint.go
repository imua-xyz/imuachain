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
	validatorPower := ms.resolveValidatorPower(ctx, evmAddr)
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
func (ms msgServer) resolveValidatorPower(ctx sdk.Context, evmAddr common.Address) int64 {
	accAddr := sdk.AccAddress(evmAddr.Bytes())

	// Production path: use operator keeper to resolve consensus key → validator power.
	if ms.operatorKeeper != nil {
		chainID := ctx.ChainID()
		found, wrappedKey, err := ms.operatorKeeper.GetOperatorConsKeyForChainID(ctx, accAddr, chainID)
		if err != nil {
			// Operator exists but key lookup failed (e.g. not registered for this chain).
			// Log and fall through to return 0.
			ctx.Logger().Debug("operator cons key lookup failed", "addr", evmAddr.Hex(), "err", err)
			return 0
		}
		if !found {
			return 0
		}

		consAddr := wrappedKey.ToConsAddr()

		// Direct lookup from dogfood validator set by consensus address.
		validator, found := ms.getImuachainValidatorByConsAddr(ctx, consAddr)
		if !found {
			return 0
		}
		return validator.Power
	}

	// Fallback for unit tests: return average power of active validators.
	return ms.fallbackAveragePower(ctx)
}

// GetImuachainValidator looks up a validator by consensus address via the dogfood keeper.
// Returns (validator, found). Panics are recovered to handle nil keepers in tests.
// imuachainValidatorInfo is a minimal struct to hold validator power.
type imuachainValidatorInfo struct {
	Power int64
}

func (ms msgServer) getImuachainValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (imuachainValidatorInfo, bool) {
	// Recover is defensive: GetAllImuachainValidators can panic when the dogfood keeper
	// is nil (unit-test wiring) or when its internal state is malformed. Surfacing the
	// panic up would halt consensus inside a msg server, so we recover and log loudly
	// (Error level) instead of swallowing — operators must see this if it ever fires
	// in production.
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error("getImuachainValidatorByConsAddr recovered from panic",
				"consAddr", consAddr.String(), "error", r)
		}
	}()
	validators := ms.GetAllImuachainValidators(ctx)
	for _, v := range validators {
		if sdk.ConsAddress(v.Address).Equals(consAddr) {
			return imuachainValidatorInfo{Power: v.Power}, true
		}
	}
	return imuachainValidatorInfo{}, false
}

func (ms msgServer) fallbackAveragePower(ctx sdk.Context) int64 {
	// Test-only path (production wires operatorKeeper); recover guards against nil
	// dogfood keeper. Log loudly so a panic firing in production is not invisible.
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error("fallbackAveragePower recovered from panic", "error", r)
		}
	}()
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
