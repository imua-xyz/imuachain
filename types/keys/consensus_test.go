package keys_test

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/stretchr/testify/suite"
)

type ConsKeyTestSuite struct {
	suite.Suite
}

func (suite *ConsKeyTestSuite) SetupTest() {
}

func TestConsKeyTestSuite(t *testing.T) {
	suite.Run(t, new(ConsKeyTestSuite))
}

func (suite *ConsKeyTestSuite) TestNewWrappedConsKeyFromJSON() {
	// success
	json := `{"@type":"/cosmos.crypto.ed25519.PubKey","key":"wtDUcGq4pGt1X0/IU1kJtfxSJAMFa7C/sBhWn8ExVOQ="}`
	key := keytypes.NewWrappedConsKeyFromJSON(json)
	suite.NotNil(key)
	suite.Equal(json, key.ToJSON())
	// fail unmarshalling
	json = `{"type":"/cosmos.crypto.ed25519.PubKey","key":"wtDUcGq4pGt1X0/IU1kJtfxSJAMFa7C/sBhWn8ExVOQ="}`
	key = keytypes.NewWrappedConsKeyFromJSON(json)
	suite.Nil(key)
	// fail key type
	json = `{"@type":"/cosmos.crypto.ed25518.PubKey","key":"wtDUcGq4pGt1X0/IU1kJtfxSJAMFa7C/sBhWn8ExVOQ="}`
	key = keytypes.NewWrappedConsKeyFromJSON(json)
	suite.Nil(key)
	// fail decoding
	json = `{"@type":"/cosmos.crypto.ed25519.PubKey","key":"wtDUcGq4pGt1X0/IU1kJtfxSJAMFa7C/sBhWn8ExVOQ=="}`
	key = keytypes.NewWrappedConsKeyFromJSON(json)
	suite.Nil(key)
}

func (suite *ConsKeyTestSuite) TestNewWrappedConsKeyFromHex() {
	// success
	hex := "0xF0F6919E522C5B97DB2C8255BFF743F9DFDDD7AD9FC37CB0C1670B480D0F9914"
	key := keytypes.NewWrappedConsKeyFromHex(hex)
	suite.NotNil(key)
	suite.Equal(hex, key.ToHex())
	// high length
	hex = "0xF0F6919E522C5B97DB2C8255BFF743F9DFDDD7AD9FC37CB0C1670B480D0F9914F0F6919E522C5B97DB2C8255BFF743F9DFDDD7AD9FC37CB0C1670B480D0F9914"
	key = keytypes.NewWrappedConsKeyFromHex(hex)
	suite.Nil(key)
	// non hex
	hex = "0xF0F6919E522C5B97DB2C8255BFF743F9DFDDD7AD9FC37CB0C1670B480D0F991G"
	key = keytypes.NewWrappedConsKeyFromHex(hex)
	suite.Nil(key)
}

func (suite *ConsKeyTestSuite) TestNewWrappedConsKeyFromTmProtoKey() {
	// success
	tmKey := &tmprotocrypto.PublicKey{
		Sum: &tmprotocrypto.PublicKey_Ed25519{
			Ed25519: ed25519.GenPrivKey().PubKey().Bytes(),
		},
	}
	key := keytypes.NewWrappedConsKeyFromTmProtoKey(tmKey)
	suite.NotNil(key)
	suite.Equal(tmKey, key.ToTmProtoKey())
	// different type failure
	tmKey = &tmprotocrypto.PublicKey{
		Sum: &tmprotocrypto.PublicKey_Secp256K1{
			Secp256K1: secp256k1.GenPrivKey().PubKey().Bytes(),
		},
	}
	key = keytypes.NewWrappedConsKeyFromTmProtoKey(tmKey)
	suite.Nil(key)
}

func (suite *ConsKeyTestSuite) TestNewWrappedConsKeyFromSdkKey() {
	// success
	sdkKey := cryptotypes.PubKey(sdked25519.GenPrivKey().PubKey())
	key := keytypes.NewWrappedConsKeyFromSdkKey(sdkKey)
	suite.NotNil(key)
	suite.Equal(sdkKey, key.ToSdkKey())
	// different type failure
	sdkKey = cryptotypes.PubKey(sdksecp256k1.GenPrivKey().PubKey())
	key = keytypes.NewWrappedConsKeyFromSdkKey(sdkKey)
	suite.Nil(key)
}
