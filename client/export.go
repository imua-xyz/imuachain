package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cometbft/cometbft/privval"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/client/keys"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v16/crypto/hd"
)

// UnsafeExportEthKeyCommand exports a key with the given name as a private key in hex format.
func UnsafeExportEthKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unsafe-export-eth-key [name]",
		Short: "**UNSAFE** Export an Ethereum private key",
		Long:  `**UNSAFE** Export an Ethereum private key unencrypted to use in dev tooling`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd).WithKeyringOptions(hd.EthSecp256k1Option())
			clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			decryptPassword := ""
			conf := true

			inBuf := bufio.NewReader(cmd.InOrStdin())
			switch clientCtx.Keyring.Backend() {
			case keyring.BackendFile:
				decryptPassword, err = input.GetPassword(
					"**WARNING this is an unsafe way to export your unencrypted private key**\nEnter key password:",
					inBuf)
			case keyring.BackendOS:
				conf, err = input.GetConfirmation(
					"**WARNING** this is an unsafe way to export your unencrypted private key, are you sure?",
					inBuf, cmd.ErrOrStderr())
			}
			if err != nil || !conf {
				return err
			}

			// Exports private key from keybase using password
			armor, err := clientCtx.Keyring.ExportPrivKeyArmor(args[0], decryptPassword)
			if err != nil {
				return err
			}

			privKey, algo, err := crypto.UnarmorDecryptPrivKey(armor, decryptPassword)
			if err != nil {
				return err
			}

			if algo != ethsecp256k1.KeyType {
				return fmt.Errorf("invalid key algorithm, got %s, expected %s", algo, ethsecp256k1.KeyType)
			}

			// Converts key to Evmos secp256k1 implementation
			ethPrivKey, ok := privKey.(*ethsecp256k1.PrivKey)
			if !ok {
				return fmt.Errorf("invalid private key type %T, expected %T", privKey, &ethsecp256k1.PrivKey{})
			}

			key, err := ethPrivKey.ToECDSA()
			if err != nil {
				return err
			}

			// Formats key for output
			privB := ethcrypto.FromECDSA(key)
			keyS := strings.ToUpper(hexutil.Encode(privB)[2:])

			fmt.Println(keyS)

			return nil
		},
	}
}

// ConsPubKeyToBytesCmd returns a command that converts a consensus public key to a byte
// representation.
func ConsPubKeyToBytesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensus-pubkey-to-bytes",
		Short: "Convert a consensus public key to a byte32 representation",
		Long:  `Convert a consensus public key to a byte32 representation for usage with Solidity contracts`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// basic stuff
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)
			// from the config, load this info
			pvKeyFile := config.PrivValidatorKeyFile()
			pvStateFile := config.PrivValidatorStateFile()
			// then load the PrivValidator file
			filePV := privval.LoadFilePV(pvKeyFile, pvStateFile)
			tmValPubKey, err := filePV.GetPubKey()
			if err != nil {
				return err
			}
			output, err := cmd.Flags().GetString(cli.OutputFlag)
			if err != nil {
				return err
			}
			outstream := cmd.OutOrStdout()
			displayConsKeyBytes(outstream, newConsKeyBytes(tmValPubKey.Bytes()), output)
			return nil
		},
	}

	return cmd
}

func displayConsKeyBytes(w io.Writer, stringer fmt.Stringer, output string) {
	var (
		err error
		out []byte
	)

	switch output {
	case keys.OutputFormatText:
		out, err = yaml.Marshal(&stringer)

	case keys.OutputFormatJSON:
		out, err = json.Marshal(&stringer)
	}

	if err != nil {
		panic(err)
	}

	_, _ = fmt.Fprintln(w, string(out))
}

type consKeyBytes struct {
	Bytes string `json:"bytes32"`
}

func (ckb consKeyBytes) String() string {
	return fmt.Sprintf("Bytes (hex): %s", ckb.Bytes)
}

func newConsKeyBytes(bz []byte) consKeyBytes {
	return consKeyBytes{Bytes: hexutil.Encode(bz)}
}
