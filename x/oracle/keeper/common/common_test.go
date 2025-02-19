package common

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -destination mock_keeper_test.go -package common github.com/imua-xyz/imuachain/x/oracle/keeper/common KeeperOracle

//go:generate mockgen -destination mock_validator_test.go -package common github.com/cosmos/cosmos-sdk/x/staking/types ValidatorI

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ko := NewMockKeeperOracle(ctrl)

	ko.EXPECT().GetLastTotalPower(gomock.Any()).Return(math.NewInt(99))

	x := ko.GetLastTotalPower(sdk.Context{})
	_ = x

	Convey("mock oracle keeper", t, func() {
		Convey("GetLastTotalPower", func() { So(x, ShouldResemble, math.NewInt(99)) })
	})
}
