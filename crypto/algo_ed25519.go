package crypto

import (
	cryptoed25519 "crypto/ed25519"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	evmoshd "github.com/evmos/evmos/v16/crypto/hd"
)

var (
	// SupportedAlgorithms is intended for testing purposes only.
	// WARNING: Ed25519 keys must not be used in SDK apps except in a Tendermint validator context.
	SupportedAlgorithms = keyring.SigningAlgoList{evmoshd.EthSecp256k1, hd.Secp256k1, Ed25519}
	Ed25519             = ed25519Algo{}
)

type ed25519Algo struct{}

func (e ed25519Algo) Name() hd.PubKeyType {
	return hd.Ed25519Type
}

// Derive derives and returns the ed25519 private key
// for ed25519, this is mainly used for test, we don't actually generate ed25519 from the path defined from slip0010, we ignore the path, and retrieve seed from mnemonic directly, then use that seed as secret to generate keys through ed25519
func (e ed25519Algo) Derive() hd.DeriveFn {
	return func(mnemonic, _, _ string) ([]byte, error) {
		// seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
		// don't do transfer to be compatible with the way used in x/genutil/utils.go.InitializeNodeValidatorFilesFromMnemonic
		seed := []byte(mnemonic)
		return ed25519.GenPrivKeyFromSecret(seed).Bytes(), nil
	}
}

// Generate will be used to import privateKey from hex through keyring, so we just return the bz as privateKey instead of seed
func (e ed25519Algo) Generate() hd.GenerateFn {
	return func(bz []byte) cryptotypes.PrivKey {
		if len(bz) != cryptoed25519.PrivateKeySize {
			panic("invalid ed25519 private key size")
		}
		//nolint:staticcheck
		return &ed25519.PrivKey{
			Key: cryptoed25519.PrivateKey(bz),
		}
	}
}

// Ed25519Option this option is mainly used for test
func Ed25519Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
	}
}
