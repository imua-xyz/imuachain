package types

import (
	"strings"
	"time"

	"cosmossdk.io/math"
	"sigs.k8s.io/yaml"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func NewValidator(operator sdk.AccAddress, pubKey cryptotypes.PubKey, description stakingtypes.Description) (Validator, error) {
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return Validator{}, err
	}

	return Validator{
		OperatorEarningsAddr:    operator.String(),
		ConsensusPubkey:         pkAny,
		Jailed:                  false,
		Status:                  stakingtypes.Bonded,
		VotingPower:             math.LegacyZeroDec(),
		DelegatorShares:         math.ZeroInt(),
		Description:             description,
		UnbondingHeight:         int64(0),
		UnbondingTime:           time.Unix(0, 0).UTC(),
		Commission:              stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		UnbondingOnHoldRefCount: 0,
		DelegatorTokens:         []DelegatorInfo{},
	}, nil
}

// String implements the Stringer interface for a Validator object.
func (v Validator) String() string {
	bz, err := codec.ProtoMarshalJSON(&v, nil)
	if err != nil {
		panic(err)
	}

	out, err := yaml.JSONToYAML(bz)
	if err != nil {
		panic(err)
	}

	return string(out)
}

// Validators is a collection of Validator
type Validators []Validator

func (v Validators) String() (out string) {
	for _, val := range v {
		out += val.String() + "\n"
	}

	return strings.TrimSpace(out)
}
