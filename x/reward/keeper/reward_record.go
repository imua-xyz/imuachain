package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/imua-xyz/imuachain/x/reward/types"
)

type rewardRecord struct {
	ctx         sdk.Context
	k           Keeper
	banker      bankkeeper.Keeper
	distributor types.Distributor
	types.Pool
	staker distrtypes.StakingKeeper
}

func newRewardRecord(ctx sdk.Context, k Keeper, banker bankkeeper.Keeper, distributor types.Distributor, _ types.Pool) *rewardRecord {
	return &rewardRecord{
		ctx:         ctx,
		k:           k,
		banker:      banker,
		distributor: distributor,
	}
}

func (p rewardRecord) getRewards(earningAddress string) (sdk.Coins, bool) {
	for _, reward := range p.Pool.Rewards {
		if reward.EarningsAddr == earningAddress {
			return reward.Coins, true
		}
	}
	return sdk.Coins{}, false
}

// Logically recording the rewards
func (p *rewardRecord) AddReward(earningAddress string, coin sdk.Coin) {
	defer func() {
		p.k.Logger(p.ctx).Debug("adding rewards in pool", "pool", p.Name, "earningAddress", earningAddress, "coin", coin.String(), "amount", coin.Amount)
		p.k.setPool(p.ctx, p.Pool)
	}()

	if coin.Amount.IsZero() {
		return
	}

	for i, reward := range p.Rewards {
		if reward.EarningsAddr == earningAddress {
			p.Rewards[i].Coins = reward.Coins.Add(coin)
			return
		}
	}

	p.Rewards = append(p.Rewards, types.Pool_Reward{
		EarningsAddr: earningAddress,
		Coins:        sdk.NewCoins(coin),
	})
}

// Allocate the rewards actually
func (p *rewardRecord) ReleaseRewards(earningAddress string) error {
	rewards, ok := p.getRewards(earningAddress)
	if !ok {
		return nil
	}

	defer p.ClearRewards(earningAddress)

	addr, err := sdk.ValAddressFromBech32(earningAddress)
	if err != nil {
		return err
	}

	v := p.staker.Validator(p.ctx, addr)
	if v == nil {
		return nil
	}

	if err := p.banker.MintCoins(p.ctx, types.ModuleName, rewards); err != nil {
		return err
	}

	if err := p.banker.SendCoinsFromModuleToModule(p.ctx, types.ModuleName, distrtypes.ModuleName, rewards); err != nil {
		return err
	}

	p.k.Logger(p.ctx).Info("releasing rewards in pool", "pool", p.Name, "earningAddress", earningAddress)

	p.distributor.AllocateTokensToValidator(
		p.ctx,
		v,
		sdk.NewDecCoinsFromCoins(rewards...),
	)

	return nil
}

func (p *rewardRecord) ClearRewards(earningAddress string) {
	for i, reward := range p.Rewards {
		if reward.EarningsAddr == earningAddress {
			p.k.Logger(p.ctx).Info("clearing rewards in pool", "pool", p.Name, "earningAddress", earningAddress)
			p.Rewards = append(p.Rewards[:i], p.Rewards[i+1:]...)
			p.k.setPool(p.ctx, p.Pool)
			return
		}
	}
}
