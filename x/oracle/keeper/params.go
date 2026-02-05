package keeper

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

const (
	startAfterBlocks = 10
	defaultInterval  = 30
)

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz != nil {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	// TODO: validation check
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)
}

func (k Keeper) RegisterNewTokenAndSetTokenFeeder(ctx sdk.Context, oInfo *types.OracleInfo) error {
	p := k.GetParams(ctx)
	if p.GetTokenIDFromAssetID(oInfo.AssetID) > 0 {
		return fmt.Errorf("assetID exists:%s", oInfo.AssetID)
	}
	chainID := uint64(0)
	for id, c := range p.Chains {
		if c.Name == oInfo.Chain.Name {
			// #nosec G115
			chainID = uint64(id)
			break
		}
	}
	if chainID == 0 {
		// add new chain
		p.Chains = append(p.Chains, &types.Chain{
			Name: oInfo.Chain.Name,
			Desc: oInfo.Chain.Desc,
		})
		// #nosec G115
		chainID = uint64(len(p.Chains) - 1)
	}
	decimalInt, err := strconv.ParseInt(oInfo.Token.Decimal, 10, 32)
	if err != nil {
		return err
	}
	intervalInt := uint64(0)
	if len(oInfo.Feeder.Interval) > 0 {
		intervalInt, err = strconv.ParseUint(oInfo.Feeder.Interval, 10, 64)
		if err != nil {
			return err
		}
	}
	if intervalInt == 0 {
		intervalInt = defaultInterval
	}

	isNST := assetstypes.IsNST(oInfo.AssetID)
	// var assetAddr string
	var clientChainID uint64

	if isNST {
		_, clientChainID, err = assetstypes.ParseID(oInfo.AssetID)
		if err != nil {
			return fmt.Errorf("invalid assetID %s: %w", oInfo.AssetID, err)
		}
	}

	defer func() {
		if !ctx.IsCheckTx() {
			k.SetParamsUpdated()
		}
	}()

	idx, has := p.HasTokenByName(oInfo.Token.Name, chainID)
	if has {
		t := p.Tokens[idx]
		t.AssetID = strings.Join([]string{t.AssetID, oInfo.AssetID}, ",")
		if !isNST {
			k.SetParams(ctx, p)
			return nil
		}
	}

	// add a new token
	p.Tokens = append(p.Tokens, &types.Token{
		Name:            oInfo.Token.Name,
		ChainID:         chainID,
		ContractAddress: oInfo.Token.Contract,
		Decimal:         int32(decimalInt), // #nosec G115
		Active:          true,
		AssetID:         oInfo.AssetID,
	})
	startBaseBlock := uint64(ctx.BlockHeight() + startAfterBlocks)
	if len(p.TokenFeeders) > 1 {
		offset := GetStartBaseBlock(startBaseBlock+1, uint64(p.MaxNonce), intervalInt, p.TokenFeeders[1:])
		// GetStartBaseBlock returns an offset relative to `firstQuotingHeight = startBaseBlock + 1`.
		//   newStartBaseBlock = startBaseBlock + offset => newStartBaseBlock + 1 = (startBaseBlock + 1) + offset
		startBaseBlock += offset
	}
	// set a tokenFeeder for the new token
	p.TokenFeeders = append(p.TokenFeeders, &types.TokenFeeder{
		// #nosec G115 // len(p.Tokens) must be positive since we just append an element for it
		TokenID:      uint64(len(p.Tokens) - 1),
		RuleID:       2,
		StartRoundID: 1,
		// #nosec G115
		StartBaseBlock: startBaseBlock,
		Interval:       intervalInt,
		// we don't end feeders for v1
		EndBlock: 0,
	})

	if isNST {
		// set a virtual token for NST to track balance change
		nstTokenName := types.NSTTokenPrefix + oInfo.Token.Name
		if _, has := p.HasTokenByName(nstTokenName, chainID); !has {
			nstAssetID := types.NSTIDPrefix + hexutil.EncodeUint64(clientChainID)
			p.Tokens = append(p.Tokens, &types.Token{
				Name:            nstTokenName,
				ChainID:         chainID,
				ContractAddress: "",
				Decimal:         int32(decimalInt), // #nosec G115
				AssetID:         nstAssetID,
			})
			// set tokenfeeder to track blanace change
			p.TokenFeeders = append(p.TokenFeeders, &types.TokenFeeder{
				// #nosec G115 // len(p.Tokens) must be positive since we just append an element for it
				TokenID:      uint64(len(p.Tokens) - 1),
				RuleID:       3,
				StartRoundID: 1,
				// #nosec G115
				StartBaseBlock: startBaseBlock,
				Interval:       intervalInt,
				// we don't end feeders for v1
				EndBlock: 0,
			})
		}
	}

	k.SetParams(ctx, p)
	return nil
}

// GetStartBaseBlock calculates the optimal offset for a new TokenFeeder's startBaseBlock
// to minimize conflicts with existing feeders' quoting windows.
//
// Calculation process:
//
//  1. Creates a conflict counter array of size 'interval' to track conflicts at each offset position
//     from firstQuotingHeight (offset 0 to interval-1).
//
//  2. For each existing TokenFeeder:
//     a. Skips feeders that have already ended (EndBlock < firstQuotingHeight)
//     b. Calculates the feeder's latest round's startBaseBlock that is <= firstQuotingHeight:
//     - If firstQuotingHeight < tf.StartBaseBlock: moves backward to find the previous round
//     - Otherwise: moves forward to find the latest round <= firstQuotingHeight
//     c. Scans all quoting windows of this feeder within [firstQuotingHeight, firstQuotingHeight+interval):
//     - Each window spans [tfStartL+1, tfStartL+window] (first quoting block to last quoting block)
//     - For each block in the window that falls within the target interval, increments the conflict
//     counter at the corresponding offset position
//     d. Continues to next round (tfStartL += tf.Interval) until reaching scanUntil or EndBlock
//
// 3. Finds the optimal offset:
//   - Returns the first offset with 0 conflicts (no overlap with existing feeders)
//   - If no zero-conflict offset exists, returns the offset with minimum conflict count
//
// Parameters:
//   - firstQuotingHeight: The target height where the new feeder should start quoting
//   - window: The size of the quoting window (number of blocks for quoting)
//   - interval: The interval between rounds for the new feeder
//   - tfs: List of existing TokenFeeders to check conflicts against
//
// Returns:
//   - An offset value (0 to interval-1) that should be added to firstQuotingHeight to calculate
//     the new feeder's startBaseBlock: startBaseBlock = firstQuotingHeight + offset - 1
//     The first quoting block will be: startBaseBlock + 1 = firstQuotingHeight + offset
//
// Example:
//
//	If firstQuotingHeight=5, window=3, interval=30, and offset=1 is returned:
//	- startBaseBlock = 5 + 1 - 1 = 5
//	- First quoting block = 5 + 1 = 6
//	- Quoting window = {6, 7, 8}
func GetStartBaseBlock(firstQuotingHeight, window, interval uint64, tfs []*types.TokenFeeder) uint64 {
	if interval == 0 {
		return 0
	}
	blocks := make([]uint64, interval)
	scanEnd := firstQuotingHeight + interval
	for _, tf := range tfs {
		if tf.EndBlock > 0 && tf.EndBlock < firstQuotingHeight {
			// tokenFeeder already ended
			continue
		}
		tfStartL := tf.StartBaseBlock
		// Calculate the first quoting block of this feeder's latest round that's <= firstQuotingHeight
		if firstQuotingHeight < tfStartL {
			rounds := (tfStartL - firstQuotingHeight) / tf.Interval
			gap := (rounds + 1) * tf.Interval
			if gap <= tfStartL {
				tfStartL -= gap
			}
		} else {
			rounds := (firstQuotingHeight - tfStartL) / tf.Interval
			tfStartL += tf.Interval * rounds
		}

		// Process all quoting windows of this feeder that overlap with our interval
		for tfStartL < scanEnd {
			qStart := tfStartL + 1
			qEnd := tfStartL + window
			if qStart < firstQuotingHeight {
				qStart = firstQuotingHeight
			}
			if qEnd >= scanEnd {
				qEnd = scanEnd - 1
			}
			for i := qStart; i <= qEnd; i++ {
				blocks[i-firstQuotingHeight]++
			}
			tfStartL += tf.Interval
			if tf.EndBlock > 0 && tfStartL >= tf.EndBlock {
				break
			}
		}
	}
	minIndex := 0
	minCount := blocks[0]
	for idx, count := range blocks {
		if count == 0 {
			return uint64(idx)
		}
		if count < minCount {
			minIndex = idx
			minCount = count
		}
	}
	return uint64(minIndex)
}
