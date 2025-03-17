package bls

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
)

const (
	// MethodFastAggregateVerify defines the ABI method name to fast verify aggregated signature and its
	// corresponding public keys
	MethodFastAggregateVerify = "fastAggregateVerify"
	// MethodVerify defines the ABI method name to verify aggregated signature and aggregated public key.
	MethodVerify              = "verify"
	MethodAggregatePubKeys    = "aggregatePubKeys"
	MethodAggregateSignatures = "aggregateSignatures"
	MethodAddTwoPubKeys       = "addTwoPubKeys"
)

// Verify checks the validity of an aggregated signature against msg and aggregated public keys.
func (p Precompile) Verify(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodVerify].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodVerify].Inputs), len(args))
	}
	sigBz, ok := args[1].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	sig, err := bls.SignatureFromBytes(sigBz)
	if err != nil {
		return nil, ErrInvalidArg
	}

	pubKeyBz, ok := args[2].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubKey, err := bls.PublicKeyFromBytes(pubKeyBz)
	if err != nil {
		return nil, ErrInvalidArg
	}

	msg, ok := args[0].([32]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	return method.Outputs.Pack(sig.Verify(pubKey, msg[:]))
}

// Verify checks the validity of an aggregated signature against msg and aggregated public keys.
func (p Precompile) FastAggregateVerify(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodFastAggregateVerify].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodFastAggregateVerify].Inputs), len(args))
	}

	sigBz, ok := args[1].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	sig, err := bls.SignatureFromBytes(sigBz)
	if err != nil {
		return nil, ErrInvalidArg
	}

	pubKeysBz, ok := args[2].([][]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubKeys := make([]common.PublicKey, len(pubKeysBz))
	for i, pubKeyBz := range pubKeysBz {
		pubKey, err := bls.PublicKeyFromBytes(pubKeyBz)
		if err != nil {
			return nil, ErrInvalidArg
		}
		pubKeys[i] = pubKey
	}

	msg, ok := args[0].([32]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	return method.Outputs.Pack(sig.FastAggregateVerify(pubKeys, msg))
}

func (p Precompile) AggregatePubKeys(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodAggregatePubKeys].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodAggregatePubKeys].Inputs), len(args))
	}

	pubKeysBz, ok := args[0].([][]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	aggregatedPubKey, err := blst.AggregatePublicKeys(pubKeysBz)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate public keys")
	}

	return method.Outputs.Pack(aggregatedPubKey.Marshal())
}

func (p Precompile) AggregateSignatures(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodAggregateSignatures].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodAggregateSignatures].Inputs), len(args))
	}
	sigsBz, ok := args[0].([][]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	aggregatedSig, err := blst.AggregateCompressedSignatures(sigsBz)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate signatures")
	}

	return method.Outputs.Pack(aggregatedSig.Marshal())
}

func (p Precompile) AddTwoPubKeys(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodAddTwoPubKeys].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodAddTwoPubKeys].Inputs), len(args))
	}
	pubKeyOneBz, ok := args[0].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubKeyTwoBz, ok := args[1].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	pubKeyOne, err := blst.PublicKeyFromBytes(pubKeyOneBz)
	if err != nil {
		return nil, ErrInvalidArg
	}
	pubKeyTwo, err := blst.PublicKeyFromBytes(pubKeyTwoBz)
	if err != nil {
		return nil, ErrInvalidArg
	}
	newPubKey := pubKeyOne.Aggregate(pubKeyTwo)

	return method.Outputs.Pack(newPubKey.Marshal())
}
