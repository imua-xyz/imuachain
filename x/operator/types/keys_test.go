package types

import (
	"testing"

	testutiltx "github.com/evmos/evmos/v16/testutil/tx"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type KeysTestSuite struct {
	suite.Suite
}

func (suite *KeysTestSuite) SetupTest() {
}

func TestKeysTestSuite(t *testing.T) {
	suite.Run(t, new(KeysTestSuite))
}

func (suite *KeysTestSuite) TestParseKeyForOperatorAndChainIDToConsKey() {
	operatorAddr := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	chainIDWithoutRevision := "imuachaintestnet_233"
	key := KeyForOperatorAndChainIDToConsKey(operatorAddr, chainIDWithoutRevision)

	parsedAddr, parsedChainID, err := ParseKeyForOperatorAndChainIDToConsKey(key[1:])
	suite.NoError(err)
	suite.Equal(operatorAddr, parsedAddr)
	suite.Equal(chainIDWithoutRevision, parsedChainID)
}

func (suite *KeysTestSuite) TestParsePrevConsKey() {
	operatorAddr := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	chainIDWithoutRevision := "imuachaintestnet_233"
	key := KeyForChainIDAndOperatorToPrevConsKey(chainIDWithoutRevision, operatorAddr)

	parsedChainID, parsedAddr, err := ParsePrevConsKey(key[1:])
	suite.NoError(err)
	suite.Equal(operatorAddr, parsedAddr)
	suite.Equal(chainIDWithoutRevision, parsedChainID)
}

func (suite *KeysTestSuite) TestParseKeyForOperatorKeyRemoval() {
	operatorAddr := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	chainIDWithoutRevision := "imuachaintestnet_233"
	key := KeyForOperatorKeyRemovalForChainID(operatorAddr, chainIDWithoutRevision)

	parsedAddr, parsedChainID, err := ParseKeyForOperatorKeyRemoval(key[1:])
	suite.NoError(err)
	suite.Equal(operatorAddr, parsedAddr)
	suite.Equal(chainIDWithoutRevision, parsedChainID)
}
