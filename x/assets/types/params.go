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

// Validate validates the set of params: 1. Check if the gateways are valid hex addresses. 2. Check for duplicates.
func (p Params) Validate() error {
	// Use map for efficient duplicate checking
	seen := make(map[string]bool)

	for _, gateway := range p.Gateways {
		// Convert to lowercase for consistent comparison
		lowercased := strings.ToLower(gateway)

		// Check if it's a valid hex address
		if !common.IsHexAddress(gateway) {
			return fmt.Errorf("invalid hex address format: %s", gateway)
		}

		// Check for duplicates
		if seen[lowercased] {
			return fmt.Errorf("duplicate gateway address: %s", gateway)
		}
		seen[lowercased] = true
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
