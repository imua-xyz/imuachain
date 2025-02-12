package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	blscommon "github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
)

func (suite *AVSTestSuite) TestOperator_pubkey() {
	operatorAddr := "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"

	privateKey, err := blst.RandKey()
	fmt.Println("privateKey:", hex.EncodeToString(privateKey.Marshal()))

	publicKey := privateKey.PublicKey()
	blsPub := &types.BlsPubKeyInfo{
		OperatorAddress: operatorAddr,
		PubKey:          publicKey.Marshal(),
		AvsAddress:      suite.avsAddress.String(),
	}
	fmt.Println("pubkey:", hex.EncodeToString(publicKey.Marshal()))
	err = suite.App.AVSManagerKeeper.SetOperatorPubKey(suite.Ctx, blsPub)
	suite.NoError(err)

	pub, err := suite.App.AVSManagerKeeper.GetOperatorPubKey(suite.Ctx, operatorAddr, suite.avsAddress.String())
	suite.NoError(err)
	suite.Equal(publicKey.Marshal(), pub.PubKey)

	taskRes := types.TaskResponse{TaskID: 1, NumberSum: big.NewInt(1000)}

	hash, _ := types.GetTaskResponseDigestEncodeByjson(taskRes)

	msgBytes := hash[:]
	fmt.Println("ResHash:", hex.EncodeToString(msgBytes))

	sig := privateKey.Sign(msgBytes)
	fmt.Println("sig:", hex.EncodeToString(sig.Marshal()))

	valid := sig.Verify(publicKey, msgBytes)
	suite.True(valid)

	valid1, _ := blst.VerifySignature(sig.Marshal(), hash, publicKey)
	suite.NoError(err)

	suite.True(valid1)

	jsonData, err := types.MarshalTaskResponse(taskRes)
	fmt.Println("jsondata:", hex.EncodeToString(jsonData))
}

func (suite *AVSTestSuite) Test_hash() {
	taskres := types.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
	jsonData, err := types.MarshalTaskResponse(taskres)
	suite.NoError(err)
	hash := crypto.Keccak256Hash(jsonData)
	taskResponseDigest := hash.Bytes()
	suite.Equal(len(taskResponseDigest), 32)
}

// For deterministic result（msg）, aggregate signatures and verify them
func (suite *AVSTestSuite) Test_bls_agg() {
	taskres := types.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
	jsonData, _ := types.MarshalTaskResponse(taskres)
	msg := crypto.Keccak256Hash(jsonData)
	privateKeys := make([]blscommon.SecretKey, 4)
	publicKeys := make([]blscommon.PublicKey, 4)

	sigs := make([]blscommon.Signature, 4)
	for i := 0; i < 4; i++ {
		privateKey, _ := blst.RandKey()
		privateKeys[i] = privateKey
		publicKeys[i] = privateKey.PublicKey()
		sigs[i] = privateKey.Sign(msg[:])
	}

	aggPublicKey := blst.AggregateMultiplePubkeys(publicKeys)

	aggSignature := blst.AggregateSignatures(sigs)

	valid := aggSignature.Verify(aggPublicKey, msg.Bytes())

	suite.True(valid, "Signature verification failed")

	valid1 := aggSignature.FastAggregateVerify(publicKeys, msg)
	suite.True(valid1, "Signature verification failed")
}

// For uncertain results, i.e. multiple msgs, the test aggregation signature verification fails
func (suite *AVSTestSuite) Test_bls_agg_uncertainMsgs() {
	privateKeys := make([]blscommon.SecretKey, 4)
	publicKeys := make([]blscommon.PublicKey, 4)
	msgs := make([][]byte, 4)
	sigs := make([]blscommon.Signature, 4)
	for i := 0; i < 4; i++ {
		privateKey, _ := blst.RandKey()
		privateKeys[i] = privateKey
		publicKeys[i] = privateKey.PublicKey()
		msgs[i] = []byte{byte(i)}
		sigs[i] = privateKey.Sign([]byte{byte(i)})
	}

	aggPublicKey := blst.AggregateMultiplePubkeys(publicKeys)

	aggSignature := blst.AggregateSignatures(sigs)

	valid := aggSignature.Verify(aggPublicKey, msgs[1])

	suite.False(valid, "Signature verification failed")

	var array32 [32]byte

	// Copy data into the 32-byte array
	copy(array32[:], msgs[1])
	valid1 := aggSignature.FastAggregateVerify(publicKeys, array32)
	suite.False(valid1, "Signature verification failed")
}
