package bls_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/precompiles/bls"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	blscommon "github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	"github.com/stretchr/testify/suite"
)

var s *PrecompileTestSuite

type PrecompileTestSuite struct {
	suite.Suite

	caller              common.Address
	msg                 [32]byte
	signature           blscommon.Signature
	publicKeys          []blscommon.PublicKey
	privateKeys         []blscommon.SecretKey
	aggregatedPublicKey blscommon.PublicKey
	precompile          *bls.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)
}

func (s *PrecompileTestSuite) SetupTest() {
	s.caller, _ = testutiltx.NewAddrKey()
	s.msg = crypto.Keccak256Hash([]byte("this is a test message"))
	s.privateKeys = make([]blscommon.SecretKey, 100)
	s.publicKeys = make([]blscommon.PublicKey, 100)

	sigs := make([]blscommon.Signature, 100)
	var err error
	for i := 0; i < 100; i++ {
		privateKey, err := blst.RandKey()
		s.Require().NoError(err, "failed to generate random private key")
		s.privateKeys[i] = privateKey
		s.publicKeys[i] = privateKey.PublicKey()
		sigs[i] = privateKey.Sign(s.msg[:])
	}

	s.aggregatedPublicKey = blst.AggregateMultiplePubkeys(s.publicKeys)
	s.signature = blst.AggregateSignatures(sigs)

	blsPrecompile, err := bls.NewPrecompile(6000)
	s.Require().NoError(err, "failed to create bls precompile")
	s.precompile = blsPrecompile

	valid := s.signature.FastAggregateVerify(s.publicKeys, s.msg)
	s.Require().True(valid, "expect verification success but got failure")
}
