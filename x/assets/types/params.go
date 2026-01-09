package types

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// DefaultGateway is the default gateway address.
	// Similar with addresses for precompiles, we could assign a default gateway address
	// in case we want to deploy them as system contracts
	DefaultGateway = "0x0000000000000000000000000000000000000901"
)

// Blacklisted gateway addresses that are not allowed to be used as gateways
var (
	// forbiddenGatewayAddresses contains addresses that are not allowed to be used as gateways
	forbiddenGatewayAddresses = []string{
		"0x0000000000000000000000000000000000000000", // Zero address
		// Ethereum standard precompiles (1-9)
		"0x0000000000000000000000000000000000000001", // ECRECOVER
		"0x0000000000000000000000000000000000000002", // SHA256
		"0x0000000000000000000000000000000000000003", // RIPEMD160
		"0x0000000000000000000000000000000000000004", // IDENTITY
		"0x0000000000000000000000000000000000000005", // MODEXP
		"0x0000000000000000000000000000000000000006", // ECADD
		"0x0000000000000000000000000000000000000007", // ECMUL
		"0x0000000000000000000000000000000000000008", // ECPAIRING
		"0x0000000000000000000000000000000000000009", // BLAKE2F
		// Imuachain specific precompiles
		"0x0000000000000000000000000000000000000400", // bech32 precompile
		"0x0000000000000000000000000000000000000804", // assets precompile
		"0x0000000000000000000000000000000000000805", // delegation precompile
		"0x0000000000000000000000000000000000000806", // reward precompile
		"0x0000000000000000000000000000000000000809", // bls precompile
		// Note: 0x0000000000000000000000000000000000000901 is the default gateway address, not blacklisted
	}
)

// NewParams creates a new Params instance.
func NewParams(gateways []string) Params {
	return Params{
		Gateways: gateways,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		[]string{DefaultGateway},
	)
}

// ValidateGatewayAddress performs basic format validation on a gateway address
func ValidateGatewayAddress(gateway string) error {
	// Basic format check (includes length check)
	if !common.IsHexAddress(gateway) {
		return fmt.Errorf("invalid hex address format: %s", gateway)
	}

	// Convert to common.Address for stricter checks
	addr := common.HexToAddress(gateway)

	// Check zero address
	if addr == (common.Address{}) {
		return fmt.Errorf("zero address is not allowed: %s", gateway)
	}

	return nil
}

// ValidateBasic validates the set of params: 1. Check if the gateways are valid hex addresses. 2. Check for duplicates.
func (p Params) ValidateBasic() error {
	// Use map for efficient duplicate checking
	seen := make(map[string]bool)

	for _, gateway := range p.Gateways {
		// Convert to lowercase for consistent comparison
		lowercased := strings.ToLower(gateway)

		// Basic gateway address validation (format only)
		if err := ValidateGatewayAddress(gateway); err != nil {
			return err
		}

		// Check for duplicates
		if seen[lowercased] {
			return fmt.Errorf("duplicate gateway address: %s", gateway)
		}
		seen[lowercased] = true
	}

	return nil
}

// ValidateGatewayBusinessRules applies business logic validation to gateway addresses
func ValidateGatewayBusinessRules(gateway string) error {
	// Check if address is in forbidden list
	for _, forbidden := range forbiddenGatewayAddresses {
		if strings.EqualFold(gateway, forbidden) {
			return ErrForbiddenGatewayAddress.Wrapf(
				"address is in forbidden list: %s", gateway)
		}
	}
	return nil
}

func (p *Params) Normalize() {
	for i, gateway := range p.Gateways {
		p.Gateways[i] = strings.ToLower(gateway)
	}
}

// ValidateHexHash validates a hex hash.
func ValidateHexHash(i interface{}) error {
	hash, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if len(common.FromHex(hash)) != common.HashLength {
		return fmt.Errorf("invalid hex hash: %s", hash)
	}
	return nil
}
