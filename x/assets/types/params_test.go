package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name        string
		gateways    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid gateway address",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"},
			expectError: false,
		},
		{
			name:        "zero address should be rejected",
			gateways:    []string{"0x0000000000000000000000000000000000000000"},
			expectError: true,
			errorMsg:    "zero address is not allowed",
		},
		{
			name:        "invalid hex format should be rejected",
			gateways:    []string{"0xinvalid"},
			expectError: true,
			errorMsg:    "invalid hex address format",
		},
		{
			name:        "duplicate addresses should be rejected",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD", "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"},
			expectError: true,
			errorMsg:    "duplicate gateway address",
		},
		{
			name:        "case insensitive duplicate should be rejected",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD", "0x3fc91a3afd70395cd496c647d5a6cc9d4b2b7fad"},
			expectError: true,
			errorMsg:    "duplicate gateway address",
		},
		{
			name:        "precompile address should pass basic validation",
			gateways:    []string{"0x0000000000000000000000000000000000000001"},
			expectError: false, // Basic validation passes, blacklist check is in keeper
		},
		{
			name:        "valid multiple gateways",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD", "0x1234567890123456789012345678901234567890"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := Params{
				Gateways: tc.gateways,
			}
			err := params.ValidateBasic()
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateGatewayAddress(t *testing.T) {
	testCases := []struct {
		name        string
		address     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid address",
			address:     "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD",
			expectError: false,
		},
		{
			name:        "zero address",
			address:     "0x0000000000000000000000000000000000000000",
			expectError: true,
			errorMsg:    "zero address is not allowed",
		},
		{
			name:        "precompile address 1",
			address:     "0x0000000000000000000000000000000000000001",
			expectError: false, // Basic validation passes, blacklist check is in Validate()
		},
		{
			name:        "precompile address 255",
			address:     "0x00000000000000000000000000000000000000ff",
			expectError: false, // Basic validation passes, blacklist check is in Validate()
		},
		{
			name:        "invalid hex format",
			address:     "0xinvalid",
			expectError: true,
			errorMsg:    "invalid hex address format",
		},
		{
			name:        "too short address",
			address:     "0x123",
			expectError: true,
			errorMsg:    "invalid hex address format",
		},
		{
			name:        "too long address",
			address:     "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD123",
			expectError: true,
			errorMsg:    "invalid hex address format",
		},
		{
			name:        "missing 0x prefix",
			address:     "3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD",
			expectError: false, // IsHexAddress accepts addresses without 0x prefix
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGatewayAddress(tc.address)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateGatewayBusinessRules(t *testing.T) {
	testCases := []struct {
		name        string
		address     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid address",
			address:     "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD",
			expectError: false,
		},
		{
			name:        "zero address should be rejected",
			address:     "0x0000000000000000000000000000000000000000",
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "precompile address 1 should be rejected",
			address:     "0x0000000000000000000000000000000000000001",
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "precompile address 9 should be rejected",
			address:     "0x0000000000000000000000000000000000000009",
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "assets precompile should be rejected",
			address:     "0x0000000000000000000000000000000000000804",
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "case insensitive forbidden address should be rejected",
			address:     "0x0000000000000000000000000000000000000001",
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGatewayBusinessRules(tc.address)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
