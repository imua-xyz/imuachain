package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
}

func (suite *GenesisTestSuite) SetupTest() {
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	sharedKey := hexutil.Encode(ed25519.GenPrivKey().PubKey().Bytes())
	operator1 := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	consAddr1 := sdk.ConsAddress(operator1)
	recordKey := hexutil.Encode(
		delegationtypes.GetUndelegationRecordKey(
			1000, // block height
			1,    // layer zero nonce
			common.BytesToHash([]byte("tx hash")).Hex(),
			operator1.String(),
		),
	)
	params := types.DefaultParams()
	params.AssetIDs = []string{"0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1"}
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
		expError string
		malleate func(*types.GenesisState)
	}{
		{
			name: "constructor",
			genState: &types.GenesisState{
				Params: params,
			},
			expPass:  false, // difference between 0 and 1 voting power
			expError: "nil last total power",
		},
		{
			name: "default",
			genState: func() *types.GenesisState {
				defaultGen := types.DefaultGenesis()
				defaultGen.Params = params
				return defaultGen
			}(),
			expPass:  false, // 0 voting power isn't permitted
			expError: "non-positive last total power",
		},
		{
			name: "NewGenesis call",
			genState: types.NewGenesis(
				params, []types.GenesisValidator{},
				[]types.EpochToOperatorAddrs{},
				[]types.EpochToConsensusAddrs{},
				[]types.EpochToUndelegationRecordKeys{},
				math.ZeroInt(),
			),
			expPass:  false, // 0 voting power isn't permitted
			expError: "non-positive last total power",
		},
		{
			name: "too many validators",
			genState: &types.GenesisState{
				Params: params,
			},
			malleate: func(gs *types.GenesisState) {
				// note the plus 1
				for i := 0; i < int(gs.Params.MaxValidators)+1; i++ {
					// generate a  new key each time
					key := hexutil.Encode(
						ed25519.GenPrivKey().PubKey().Bytes(),
					)
					gs.ValSet = append(gs.ValSet,
						types.GenesisValidator{
							PublicKey: key,
							Power:     5,
						},
					)
				}
				gs.LastTotalPower = math.NewInt(
					int64(len(gs.ValSet) * 5),
				)
			},
			expPass:  false,
			expError: "too many validators",
		},
		{
			name: "duplicate keys",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
					{
						PublicKey: sharedKey,
						Power:     10,
					},
				},
				LastTotalPower: math.NewInt(10),
			},
			expPass:  false,
			expError: "duplicate public key",
		},
		{
			name: "key with missing 0x prefix",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						// remove 2 chars and add 2 chars
						PublicKey: sharedKey[2:] + "ab",
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
			},
			expPass:  false,
			expError: "invalid public key",
		},
		{
			// also covers empty key
			name: "key with the wrong length",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey + "ab",
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
			},
			expPass:  false,
			expError: "invalid public key",
		},
		{
			name: "non hex key",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						// replace last 2 chars with non-hex
						PublicKey: sharedKey[:64] + "ss",
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
			},
			expPass:  false,
			expError: "invalid public key",
		},
		{
			name: "negative vote power",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     -1,
					},
				},
				LastTotalPower: math.NewInt(1),
			},
			expPass:  false,
			expError: "less than min self delegation",
		},
		{
			name: "valid genesis with one validator",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
			},
			expPass: true,
		},
		{
			name: "duplicate epoch in expiries",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 2,
						OperatorAccAddrs: []string{
							operator1.String(),
						},
					},
					{
						Epoch: 2,
						OperatorAccAddrs: []string{
							operator1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate epoch",
		},
		{
			name: "epoch 1 for expiries",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 1,
						OperatorAccAddrs: []string{
							operator1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "should be > 1",
		},
		{
			name: "empty operator addrs for expiry epoch",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 2,
					},
				},
			},
			expPass:  false,
			expError: "empty operator addresses for epoch",
		},
		{
			name: "duplicate addrs for expiry epoch",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 2,
						OperatorAccAddrs: []string{
							operator1.String(),
							operator1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate operator address",
		},
		{
			name: "invalid addr for expiry epoch",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 2,
						OperatorAccAddrs: []string{
							"invalid address",
						},
					},
				},
			},
			expPass:  false,
			expError: "invalid operator address",
		},
		{
			name: "valid with expiries",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				OptOutExpiries: []types.EpochToOperatorAddrs{
					{
						Epoch: 2,
						OperatorAccAddrs: []string{
							operator1.String(),
						},
					},
				},
			},
			expPass: true,
		},
		{
			name: "duplicate epoch in pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 2,
						ConsAddrs: []string{
							consAddr1.String(),
						},
					},
					{
						Epoch: 2,
						ConsAddrs: []string{
							consAddr1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate epoch",
		},
		{
			name: "epoch 1 for pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 1,
						ConsAddrs: []string{
							consAddr1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "should be > 1",
		},
		{
			name: "empty cons addrs for pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 2,
					},
				},
			},
			expPass:  false,
			expError: "empty consensus addresses for epoch",
		},
		{
			name: "duplicate cons addrs for pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 2,
						ConsAddrs: []string{
							consAddr1.String(),
							consAddr1.String(),
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate consensus address",
		},
		{
			name: "valid with pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 2,
						ConsAddrs: []string{
							consAddr1.String(),
						},
					},
				},
			},
			expPass: true,
		},
		{
			name: "invalid cons addrs for pruning",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				ConsensusAddrsToPrune: []types.EpochToConsensusAddrs{
					{
						Epoch: 2,
						ConsAddrs: []string{
							"invalid cons address",
						},
					},
				},
			},
			expPass:  false,
			expError: "invalid consensus address",
		},
		{
			name: "duplicate epoch in undelegations",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							recordKey,
						},
					},
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							recordKey,
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate epoch",
		},
		{
			name: "epoch 1 for undelegations",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 1,
						UndelegationRecordKeys: []string{
							recordKey,
						},
					},
				},
			},
			expPass:  false,
			expError: "should be > 1",
		},
		{
			name: "empty record keys for undelegations",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
					},
				},
			},
			expPass:  false,
			expError: "empty record keys for epoch",
		},
		{
			name: "duplicate undelegation record keys",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							recordKey,
							recordKey,
						},
					},
				},
			},
			expPass:  false,
			expError: "duplicate record key",
		},
		{
			name: "valid with undelegation record key",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							recordKey,
						},
					},
				},
			},
			expPass: true,
		},
		{
			name: "undelegation record key: not hex",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							"not hex",
						},
					},
				},
			},
			expPass:  false,
			expError: "invalid record key (non hex)",
		},
		{
			name: "undelegation record key: can't parse",
			genState: &types.GenesisState{
				Params: params,
				ValSet: []types.GenesisValidator{
					{
						PublicKey: sharedKey,
						Power:     5,
					},
				},
				LastTotalPower: math.NewInt(5),
				UndelegationMaturities: []types.EpochToUndelegationRecordKeys{
					{
						Epoch: 2,
						UndelegationRecordKeys: []string{
							"0x1234",
						},
					},
				},
			},
			expPass:  false,
			expError: "invalid record key (parse)",
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.malleate != nil {
			tc.malleate(tc.genState)
		}
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
			suite.Require().Condition(func() bool {
				return len(tc.expError) > 0
			}, "expError not set for %s", tc.name)
			suite.Require().ErrorContains(err, tc.expError, tc.name)
		}
		// fmt.Println(tc.name, err)
	}
}
