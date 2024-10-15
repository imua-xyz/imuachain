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
	MethodAggregatePubkeys    = "aggregatePubkeys"
	MethodAggregateSignatures = "aggregateSignatures"
	MethodAddTwoPubkeys       = "addTwoPubkeys"
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

	pubkeyBz, ok := args[2].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubkey, err := bls.PublicKeyFromBytes(pubkeyBz)
	if err != nil {
		return nil, ErrInvalidArg
	}

	msg, ok := args[0].([32]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	return method.Outputs.Pack(sig.Verify(pubkey, msg[:]))
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

	pubkeysBz, ok := args[2].([][]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubkeys := make([]common.PublicKey, len(pubkeysBz))
	for i, pubkeyBz := range pubkeysBz {
		pubkey, err := bls.PublicKeyFromBytes(pubkeyBz)
		if err != nil {
			return nil, ErrInvalidArg
		}
		pubkeys[i] = pubkey
	}

	msg, ok := args[0].([32]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	return method.Outputs.Pack(sig.FastAggregateVerify(pubkeys, msg))
}

func (p Precompile) AggregatePubkeys(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodAggregatePubkeys].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodAggregatePubkeys].Inputs), len(args))
	}

	pubkeysBz, ok := args[0].([][]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	aggregatedPubkey, err := blst.AggregatePublicKeys(pubkeysBz)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate public keys")
	}

	return method.Outputs.Pack(aggregatedPubkey.Marshal())
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

func (p Precompile) AddTwoPubkeys(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodAddTwoPubkeys].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodAddTwoPubkeys].Inputs), len(args))
	}
	pubkeyOneBz, ok := args[0].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}
	pubkeyTwoBz, ok := args[1].([]byte)
	if !ok {
		return nil, ErrInvalidArg
	}

	pubkeyOne, err := blst.PublicKeyFromBytes(pubkeyOneBz)
	if err != nil {
		return nil, ErrInvalidArg
	}
	pubkeyTwo, err := blst.PublicKeyFromBytes(pubkeyTwoBz)
	if err != nil {
		return nil, ErrInvalidArg
	}
	newPubkey := pubkeyOne.Aggregate(pubkeyTwo)

	return method.Outputs.Pack(newPubkey.Marshal())
}
