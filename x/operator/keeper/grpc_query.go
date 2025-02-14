package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ExocoreNetwork/exocore/utils"

	"github.com/ethereum/go-ethereum/common"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"

	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

// QueryOperatorInfo queries the operator information for the given address.
func (k *Keeper) QueryOperatorInfo(
	ctx context.Context, req *types.GetOperatorInfoReq,
) (*types.OperatorInfo, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.OperatorInfo(c, req.OperatorAddr)
}

// QueryAllOperators queries all operators on the chain.
func (k *Keeper) QueryAllOperators(
	goCtx context.Context, req *types.QueryAllOperatorsRequest,
) (*types.QueryAllOperatorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := make([]string, 0)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatorInfo)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, _ []byte) error {
		addr := sdk.AccAddress(key)
		res = append(res, addr.String())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllOperatorsResponse{
		OperatorAccAddrs: res,
		Pagination:       pageRes,
	}, nil
}

// QueryOperatorConsKeyForChainID queries the consensus key for the operator on the given chain.
func (k *Keeper) QueryOperatorConsKeyForChainID(
	goCtx context.Context,
	req *types.QueryOperatorConsKeyRequest,
) (*types.QueryOperatorConsKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(req.OperatorAccAddr)
	if err != nil {
		return nil, err
	}
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(req.Chain)
	found, key, err := k.GetOperatorConsKeyForChainID(
		ctx, addr, chainIDWithoutRevision,
	)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("no key assigned")
	}
	return &types.QueryOperatorConsKeyResponse{
		PublicKey: *key.ToTmProtoKey(),
		OptingOut: k.IsOperatorRemovingKeyFromChainID(ctx, addr, chainIDWithoutRevision),
	}, nil
}

// QueryOperatorConsAddressForChainID queries the consensus address for the operator on
// the given chain.
func (k Keeper) QueryOperatorConsAddressForChainID(
	goCtx context.Context,
	req *types.QueryOperatorConsAddressRequest,
) (*types.QueryOperatorConsAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(req.OperatorAccAddr)
	if err != nil {
		return nil, err
	}
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(req.Chain)
	found, wrappedKey, err := k.GetOperatorConsKeyForChainID(
		ctx, addr, chainIDWithoutRevision,
	)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("no key assigned")
	}
	return &types.QueryOperatorConsAddressResponse{
		ConsAddr:  wrappedKey.ToConsAddr().String(),
		OptingOut: k.IsOperatorRemovingKeyFromChainID(ctx, addr, chainIDWithoutRevision),
	}, nil
}

// QueryAllOperatorConsKeysByChainID queries all operators for the given chain and returns
// their consensus keys.
func (k Keeper) QueryAllOperatorConsKeysByChainID(
	goCtx context.Context,
	req *types.QueryAllOperatorConsKeysByChainIDRequest,
) (*types.QueryAllOperatorConsKeysByChainIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := make([]*types.OperatorConsKeyPair, 0)
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(req.Chain)
	chainPrefix := types.ChainIDAndAddrKey(
		types.BytePrefixForChainIDAndOperatorToConsKey,
		chainIDWithoutRevision, nil,
	)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), chainPrefix)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		addr := sdk.AccAddress(key)
		ret := &tmprotocrypto.PublicKey{}
		// don't use MustUnmarshal to not panic for queries
		if err := ret.Unmarshal(value); err != nil {
			return err
		}
		res = append(res, &types.OperatorConsKeyPair{
			OperatorAccAddr: addr.String(),
			PublicKey:       ret,
			OptingOut:       k.IsOperatorRemovingKeyFromChainID(ctx, addr, chainIDWithoutRevision),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllOperatorConsKeysByChainIDResponse{
		OperatorConsKeys: res,
		Pagination:       pageRes,
	}, nil
}

// QueryAllOperatorConsAddrsByChainID queries all operators for the given chain and returns
// their consensus addresses.
func (k Keeper) QueryAllOperatorConsAddrsByChainID(
	goCtx context.Context,
	req *types.QueryAllOperatorConsAddrsByChainIDRequest,
) (*types.QueryAllOperatorConsAddrsByChainIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := make([]*types.OperatorConsAddrPair, 0)
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(req.Chain)
	chainPrefix := types.ChainIDAndAddrKey(
		types.BytePrefixForChainIDAndOperatorToConsKey,
		chainIDWithoutRevision, nil,
	)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), chainPrefix)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		addr := sdk.AccAddress(key)
		ret := &tmprotocrypto.PublicKey{}
		// don't use MustUnmarshal to not panic for queries
		if err := ret.Unmarshal(value); err != nil {
			return err
		}
		wrappedKey := keytypes.NewWrappedConsKeyFromTmProtoKey(ret)
		if wrappedKey == nil {
			return types.ErrInvalidConsKey
		}
		res = append(res, &types.OperatorConsAddrPair{
			OperatorAccAddr: addr.String(),
			ConsAddr:        wrappedKey.ToConsAddr().String(),
			OptingOut:       k.IsOperatorRemovingKeyFromChainID(ctx, addr, chainIDWithoutRevision),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllOperatorConsAddrsByChainIDResponse{
		OperatorConsAddrs: res,
		Pagination:        pageRes,
	}, nil
}

func (k *Keeper) QueryOperatorUSDValue(ctx context.Context, req *types.QueryOperatorUSDValueRequest) (*types.QueryOperatorUSDValueResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	optedUSDValues, err := k.GetOperatorOptedUSDValue(c, req.AvsAddress, req.OperatorAddr)
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorUSDValueResponse{
		USDValues: &optedUSDValues,
	}, nil
}

func (k *Keeper) QueryAVSUSDValue(ctx context.Context, req *types.QueryAVSUSDValueRequest) (*types.DecValueField, error) {
	c := sdk.UnwrapSDKContext(ctx)
	usdValue, err := k.GetAVSUSDValue(c, req.AVSAddress)
	if err != nil {
		return nil, err
	}
	return &types.DecValueField{
		Amount: usdValue,
	}, nil
}

func (k *Keeper) QueryOperatorSlashInfo(goCtx context.Context, req *types.QueryOperatorSlashInfoRequest) (*types.QueryOperatorSlashInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := make([]*types.OperatorSlashInfoByID, 0)

	slashPrefix := utils.AppendMany(types.KeyPrefixOperatorSlashInfo, assetstype.GetJoinedStoreKeyForPrefix(req.OperatorAddr, strings.ToLower(req.AvsAddress)))
	store := prefix.NewStore(ctx.KVStore(k.storeKey), slashPrefix)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		ret := &types.OperatorSlashInfo{}
		// don't use MustUnmarshal to not panic for queries
		if err := ret.Unmarshal(value); err != nil {
			return err
		}

		res = append(res, &types.OperatorSlashInfoByID{
			SlashID: string(key),
			Info:    ret,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorSlashInfoResponse{
		AllSlashInfo: res,
		Pagination:   pageRes,
	}, nil
}

func (k *Keeper) QueryAllOperatorsWithOptInAVS(goCtx context.Context, req *types.QueryAllOperatorsByOptInAVSRequest) (*types.QueryAllOperatorsByOptInAVSResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	operatorList, err := k.GetOptedInOperatorListByAVS(ctx, req.Avs)
	if err != nil {
		return nil, err
	}
	return &types.QueryAllOperatorsByOptInAVSResponse{
		OperatorList: operatorList,
	}, nil
}

func (k *Keeper) QueryAllAVSsByOperator(goCtx context.Context, req *types.QueryAllAVSsByOperatorRequest) (*types.QueryAllAVSsByOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	avsList, err := k.GetOptedInAVSForOperator(ctx, req.Operator)
	if err != nil {
		return nil, err
	}
	return &types.QueryAllAVSsByOperatorResponse{
		AvsList: avsList,
	}, nil
}

func (k *Keeper) QueryOptInfo(goCtx context.Context, req *types.QueryOptInfoRequest) (*types.OptedInfo, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return k.GetOptedInfo(ctx, req.OperatorAddr, req.AvsAddress)
}

func (k *Keeper) Validators(c context.Context, req *types.QueryValidatorsRequest) (*types.QueryValidatorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	vals := make([]types.Validator, 0)
	var chainIDWithoutRevision string

	if len(req.Chain) == 0 {
		chainIDWithoutRevision = avstypes.ChainIDWithoutRevision(ctx.ChainID())
	} else {
		chainIDWithoutRevision = avstypes.ChainIDWithoutRevision(req.Chain)
	}
	chainPrefix := types.ChainIDAndAddrKey(
		types.BytePrefixForChainIDAndOperatorToConsKey,
		chainIDWithoutRevision, nil,
	)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), chainPrefix)
	pageRes, err := query.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		ret := &tmprotocrypto.PublicKey{}
		// don't use MustUnmarshal to not panic for queries
		if err := ret.Unmarshal(value); err != nil {
			return status.Errorf(codes.Internal, "failed to unmarshal public key: %v", err)
		}
		wrappedKey := keytypes.NewWrappedConsKeyFromTmProtoKey(ret)
		if wrappedKey == nil {
			return status.Error(codes.Internal, "invalid consensus key")
		}
		val, found := k.GetValidatorByConsAddrForChainID(
			ctx, wrappedKey.ToConsAddr(), avstypes.ChainIDWithoutRevision(ctx.ChainID()),
		)
		if found {
			vals = append(vals, val)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryValidatorsResponse{Validators: vals, Pagination: pageRes}, nil
}

// Validator queries validator info for given validator address
func (k *Keeper) Validator(c context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.ValidatorAccAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}
	ctx := sdk.UnwrapSDKContext(c)

	var chainIDWithoutRevision string

	if len(req.Chain) == 0 {
		chainIDWithoutRevision = avstypes.ChainIDWithoutRevision(ctx.ChainID())
	} else {
		chainIDWithoutRevision = avstypes.ChainIDWithoutRevision(req.Chain)
	}
	accAddr, err := sdk.AccAddressFromBech32(req.ValidatorAccAddr)
	if err != nil {
		return nil, err
	}

	found, wrappedKey, err := k.GetOperatorConsKeyForChainID(
		ctx, accAddr, avstypes.ChainIDWithoutRevision(chainIDWithoutRevision),
	)

	if !found || err != nil || wrappedKey == nil {
		if err != nil {
			return nil, err
		}
		return nil, status.Errorf(codes.NotFound, "validator %s not found", req.ValidatorAccAddr)
	}

	val, found := k.GetValidatorByConsAddrForChainID(
		ctx, wrappedKey.ToConsAddr(), avstypes.ChainIDWithoutRevision(ctx.ChainID()),
	)
	if !found {
		return nil, status.Errorf(codes.NotFound, "validator %s not found", req.ValidatorAccAddr)
	}

	return &types.QueryValidatorResponse{Validator: val}, nil
}

func (k *Keeper) QuerySnapshotHelper(goCtx context.Context, req *types.QuerySnapshotHelperRequest) (*types.SnapshotHelper, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	snapshotHelper, err := k.GetSnapshotHelper(ctx, req.Avs)
	if err != nil {
		return nil, err
	}
	return &snapshotHelper, nil
}

func (k *Keeper) QuerySpecifiedSnapshot(goCtx context.Context, req *types.QuerySpecifiedSnapshotRequest) (*types.VotingPowerSnapshotKeyHeight, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	findHeight, snapshot, err := k.LoadVotingPowerSnapshot(ctx, req.Avs, req.Height)
	if err != nil {
		return nil, err
	}
	return &types.VotingPowerSnapshotKeyHeight{
		SnapshotKeyHeight: findHeight,
		Snapshot:          snapshot,
	}, nil
}

func (k *Keeper) QueryAllSnapshot(goCtx context.Context, req *types.QueryAllSnapshotRequest) (*types.QueryAllSnapshotResponse, error) {
	if !common.IsHexAddress(req.Avs) {
		return nil, fmt.Errorf("invalid AVS address format: %s", req.Avs)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := make([]*types.VotingPowerSnapshotKeyHeight, 0)

	snapshotPrefix := utils.AppendMany(types.KeyPrefixVotingPowerSnapshot, common.HexToAddress(req.Avs).Bytes())
	store := prefix.NewStore(ctx.KVStore(k.storeKey), snapshotPrefix)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, value []byte) error {
		ret := &types.VotingPowerSnapshot{}
		// don't use MustUnmarshal to not panic for queries
		if err := ret.Unmarshal(value); err != nil {
			return err
		}
		height := binary.BigEndian.Uint64(key)
		if height > math.MaxInt64 {
			return fmt.Errorf("height exceeds int64 max value: %d", height)
		}
		res = append(res, &types.VotingPowerSnapshotKeyHeight{
			SnapshotKeyHeight: int64(height),
			Snapshot:          ret,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllSnapshotResponse{
		Snapshots:  res,
		Pagination: pageRes,
	}, nil
}
