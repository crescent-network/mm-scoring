package cmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	utils "github.com/crescent-network/crescent/v4/types"
	marketmakertypes "github.com/crescent-network/crescent/v4/x/marketmaker/types"
)

var FlagManualParams = true
var ManualParams = marketmakertypes.Params{
	IncentiveBudgetAddress: marketmakertypes.DefaultIncentiveBudgetAddress.String(),
	DepositAmount:          marketmakertypes.DefaultDepositAmount,
	Common:                 marketmakertypes.DefaultCommon,
	IncentivePairs: []marketmakertypes.IncentivePair{
		{
			PairId:          12, // WETH.grv / USDC.grv
			UpdateTime:      utils.ParseTime("2023-05-01T00:00:00Z"),
			IncentiveWeight: sdk.MustNewDecFromStr("0.5"),
			MaxSpread:       sdk.MustNewDecFromStr("0.006"),
			MinWidth:        sdk.MustNewDecFromStr("0.001"),
			MinDepth:        sdk.NewInt(600000000000000000),
		},
		{
			PairId:          20, // ATOM / USDC.grv
			UpdateTime:      utils.ParseTime("2023-05-01T00:00:00Z"),
			IncentiveWeight: sdk.MustNewDecFromStr("0.5"),
			MaxSpread:       sdk.MustNewDecFromStr("0.0012"),
			MinWidth:        sdk.MustNewDecFromStr("0.002"),
			MinDepth:        sdk.NewInt(100000000),
		},
		// TODO: add other pairs
	},
}

var FlagManualMarketMakers = true
var ManualMarketMakers = []marketmakertypes.MarketMaker{
	{
		Address:  "cre123fcfc6q4lzhgem6cs2d34andlslpejuhy0540",
		PairId:   12,
		Eligible: true,
	},
	{
		Address:  "cre123fcfc6q4lzhgem6cs2d34andlslpejuhy0540",
		PairId:   20,
		Eligible: true,
	},
	// TODO: add other market makers
}
