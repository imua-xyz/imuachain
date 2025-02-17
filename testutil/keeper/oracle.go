package keeper

import (
	"testing"
	"time"

	"github.com/ExocoreNetwork/exocore/cmd/config"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"

	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	delegationkeeper "github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	dogfoodkeeper "github.com/ExocoreNetwork/exocore/x/dogfood/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/stretchr/testify/require"
)

func OracleKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	// force to initialize config at the start of each test
	// otherwise, it will cause test failure because of different prefix
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"OracleParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		dogfoodkeeper.Keeper{},
		delegationkeeper.Keeper{},
		assetskeeper.Keeper{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		slashingkeeper.Keeper{},
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{
		Time: time.Now().UTC(),
	}, false, log.NewNopLogger())

	// Initialize params
	p4Test := types.DefaultParams()
	p4Test.Chains = append(p4Test.Chains, &types.Chain{Name: "Ethereum", Desc: "-"})
	p4Test.Tokens = append(p4Test.Tokens, &types.Token{
		Name:            "ETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         18,
		Active:          true,
		AssetID:         "0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1",
	})
	p4Test.Sources = append(p4Test.Sources, &types.Source{
		Name: "Chainlink",
		Entry: &types.Endpoint{
			Offchain: map[uint64]string{0: ""},
		},
		Valid:         true,
		Deterministic: true,
	})
	p4Test.Rules = append(p4Test.Rules, &types.RuleSource{
		// all sources math
		SourceIDs: []uint64{0},
	})
	p4Test.TokenFeeders = append(p4Test.TokenFeeders, &types.TokenFeeder{
		TokenID:        1,
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 1,
		Interval:       10,
	})
	k.SetParams(ctx, p4Test)

	return &k, ctx
}
