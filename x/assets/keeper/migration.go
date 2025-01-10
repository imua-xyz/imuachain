package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	UpgradedGateway = "0x0000000000000000000000000000000000000911"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) MigrateForTest(ctx sdk.Context) error {
	params, err := m.keeper.GetParams(ctx)
	if err != nil {
		return err
	}
	params.Gateways = []string{UpgradedGateway}
	return m.keeper.SetParams(ctx, params)
}
