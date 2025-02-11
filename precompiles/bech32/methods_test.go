package bech32_test

import (
	"fmt"

	"github.com/ExocoreNetwork/exocore/precompiles/bech32"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	cmn "github.com/evmos/evmos/v16/precompiles/common"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
)

func (s *Bech32PrecompileSuite) TestHexToBech32() {
	s.SetupTest()

	method := s.precompile.Methods[bech32.MethodHexToBech32]
	sampleAddress := testutiltx.GenerateAddress()
	samplePrefix := "pre"
	sampleAddressConverted, err := sdk.Bech32ifyAddressBytes(samplePrefix, sampleAddress.Bytes())
	s.Require().NoError(err)

	testCases := []struct {
		name              string
		args              []interface{}
		expErr            bool
		errContains       string
		postCheckIfNotErr func([]byte)
	}{
		{
			name:        "invalid number of args",
			args:        []interface{}{},
			expErr:      true,
			errContains: fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, len(method.Inputs), 0),
		},
		{
			name: "invalid hex address",
			args: []interface{}{
				"", samplePrefix,
			},
			expErr:      true,
			errContains: "invalid hex address",
		},
		{
			name: "empty bech32 prefix",
			args: []interface{}{
				sampleAddress, "",
			},
			expErr:      true,
			errContains: "empty bech32 prefix provided, expected a non-empty string",
		},
		{
			name: "valid",
			args: []interface{}{
				sampleAddress, samplePrefix,
			},
			expErr: false,
			postCheckIfNotErr: func(data []byte) {
				args, err := s.precompile.Unpack(method.Name, data)
				s.Require().NoError(err)
				s.Require().Len(args, 1)
				addr, ok := args[0].(string)
				s.Require().True(ok)
				s.Require().Equal(sampleAddressConverted, addr)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.precompile.HexToBech32(&method, tc.args)
			if tc.expErr {
				s.Require().Nil(tc.postCheckIfNotErr)
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().Empty(tc.errContains)
				s.Require().NoError(err)
				s.Require().NotNil(res)
				if tc.postCheckIfNotErr != nil {
					tc.postCheckIfNotErr(res)
				}
			}
		})
	}
}

func (s *Bech32PrecompileSuite) TestBech32ToHex() {
	s.SetupTest()

	method := s.precompile.Methods[bech32.MethodBech32ToHex]
	sampleAddressHex := testutiltx.GenerateAddress()
	sampleAddress := sdk.AccAddress(sampleAddressHex.Bytes()).String()

	testCases := []struct {
		name              string
		args              []interface{}
		expErr            bool
		errContains       string
		postCheckIfNotErr func([]byte)
	}{
		{
			name:        "invalid number of args",
			args:        []interface{}{},
			expErr:      true,
			errContains: fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, len(method.Inputs), 0),
		},
		{
			name: "invalid bech32 address",
			args: []interface{}{
				1,
			},
			expErr:      true,
			errContains: "invalid bech32 address",
		},
		{
			name: "no separator",
			args: []interface{}{
				"hellothisisaninvalidaddress",
			},
			expErr:      true,
			errContains: "invalid bech32 address (no separator)",
		},
		{
			name: "valid",
			args: []interface{}{
				sampleAddress,
			},
			expErr: false,
			postCheckIfNotErr: func(data []byte) {
				args, err := s.precompile.Unpack(method.Name, data)
				s.Require().NoError(err)
				s.Require().Len(args, 1)
				addr, ok := args[0].(common.Address)
				s.Require().True(ok)
				s.Require().Equal(sampleAddressHex, addr)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := s.precompile.Bech32ToHex(&method, tc.args)
			if tc.expErr {
				s.Require().Nil(tc.postCheckIfNotErr)
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().Empty(tc.errContains)
				s.Require().NoError(err)
				s.Require().NotNil(res)
				if tc.postCheckIfNotErr != nil {
					tc.postCheckIfNotErr(res)
				}
			}
		})
	}
}
