package cv2

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	cv2keeper "github.com/ExocoreNetwork/exocore/x/oracle/migrations/cv2/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateV1ToV2(ctx sdk.Context, k keeper.Keeper) error {
	cv2keeper.MigrationParams(ctx, k)
	return nil
}
