package cmd_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	utils "github.com/crescent-network/crescent/v3/types"
	"github.com/crescent-network/crescent/v3/x/liquidity/types"
	"github.com/stretchr/testify/suite"

	"github.com/crescent-network/mm-scoring/cmd"
)

type OrderTestSuite struct {
	suite.Suite

	params cmd.Params
}

func TestOrderTestSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

func (suite *OrderTestSuite) SetupTest() {
	suite.params = cmd.Params{
		RemnantThreshold: sdk.MustNewDecFromStr("0.5"),
		AskQ1:            sdk.NewInt(5000000000),
		// TODO: MaxSpread, MinWidth, MinDepth, etc,
	}
}

func (suite *OrderTestSuite) TestGetResult() {
	for _, tc := range []struct {
		Name               string
		Orders             []types.Order
		MidPrice           sdk.Dec
		Spread             sdk.Dec
		AskWidth           sdk.Dec
		BidWidth           sdk.Dec
		AskQuantity        sdk.Int
		BidQuantity        sdk.Int
		AskMaxPrice        sdk.Dec
		AskMinPrice        sdk.Dec
		BidMaxPrice        sdk.Dec
		BidMinPrice        sdk.Dec
		BidCount           int
		AskCount           int
		RemCount           int
		InvalidStatusCount int
		TotalCount         int
	}{
		{
			Name: "case1",
			Orders: []types.Order{
				{ // included
					Id:         478556,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("1.300000000000000000"),
					Amount:     sdk.NewInt(10000000000),
					OpenAmount: sdk.NewInt(10000000000),
					BatchId:    399117,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:22.760931916Z"),
					Status:     types.OrderStatusNotMatched, // 2
				},
				{ // included
					Id:         478557,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("1.200000000000000000"),
					Amount:     sdk.NewInt(10000000000),
					OpenAmount: sdk.NewInt(10000000000),
					BatchId:    399117,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:22.760931916Z"),
					Status:     types.OrderStatusNotMatched, // 2
				},
				{ // included
					Id:         238058,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("1.100000000000000000"),
					Amount:     sdk.NewInt(20000000000),
					OpenAmount: sdk.NewInt(20000000000),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusNotMatched, // 2
				},
				{ // included
					Id:         238059,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("1.000000000000000000"),
					Amount:     sdk.NewInt(30000000000),
					OpenAmount: sdk.NewInt(30000000000),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         238061,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("1.150000000000000000"),
					Amount:     sdk.NewInt(20000000000),
					OpenAmount: sdk.NewInt(4900000000),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusPartiallyMatched, // 2
				},
				{ // included
					Id:         238062,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("1.140000000000000000"),
					Amount:     sdk.NewInt(8000000000),
					OpenAmount: sdk.NewInt(4100000000),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusPartiallyMatched, // 2
				},
				{
					Id:         238063,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("0.900000000000000000"),
					Amount:     sdk.NewInt(10000000000),
					OpenAmount: sdk.NewInt(10000000000),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusCanceled,
				},
				{
					Id:         238064,
					PairId:     1,
					MsgHeight:  478557,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 1
					Price:      sdk.MustNewDecFromStr("0.900000000000000000"),
					Amount:     sdk.NewInt(10000000000),
					OpenAmount: sdk.NewInt(0),
					BatchId:    399118,
					ExpireAt:   utils.ParseTime("2022-05-16T13:41:28.708745379Z"),
					Status:     types.OrderStatusCompleted,
				},
			},
			MidPrice:           sdk.MustNewDecFromStr("1.170000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.051282051282051282"),
			AskWidth:           sdk.MustNewDecFromStr("0.085470085470085470"),
			BidWidth:           sdk.MustNewDecFromStr("0.119658119658119658"),
			AskQuantity:        sdk.NewInt(20000000000),
			BidQuantity:        sdk.NewInt(54100000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("1.300000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("1.200000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("1.140000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("1.000000000000000000"),
			BidCount:           3,
			AskCount:           2,
			RemCount:           1,
			InvalidStatusCount: 2,
			TotalCount:         8,
		},
		{
			Name: "case2 - spec example",
			Orders: []types.Order{
				{
					Id:         478556,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.990000000000000000"),
					Amount:     sdk.NewInt(50000000),
					OpenAmount: sdk.NewInt(50000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478557,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.980000000000000000"),
					Amount:     sdk.NewInt(50000000),
					OpenAmount: sdk.NewInt(50000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478558,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.970000000000000000"),
					Amount:     sdk.NewInt(50000000),
					OpenAmount: sdk.NewInt(50000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478559,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.960000000000000000"),
					Amount:     sdk.NewInt(50000000),
					OpenAmount: sdk.NewInt(50000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478560,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 2
					Price:      sdk.MustNewDecFromStr("9.930000000000000000"),
					Amount:     sdk.NewInt(40000000),
					OpenAmount: sdk.NewInt(40000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478561,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 2
					Price:      sdk.MustNewDecFromStr("9.920000000000000000"),
					Amount:     sdk.NewInt(40000000),
					OpenAmount: sdk.NewInt(40000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478562,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 2
					Price:      sdk.MustNewDecFromStr("9.910000000000000000"),
					Amount:     sdk.NewInt(40000000),
					OpenAmount: sdk.NewInt(40000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
				{
					Id:         478563,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionBuy, // 2
					Price:      sdk.MustNewDecFromStr("9.900000000000000000"),
					Amount:     sdk.NewInt(40000000),
					OpenAmount: sdk.NewInt(40000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
			},
			MidPrice:           sdk.MustNewDecFromStr("9.945000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.003016591251885370"), // TODO: Q, truncated to 0.0030 or actual
			AskWidth:           sdk.MustNewDecFromStr("0.003016591251885370"),
			BidWidth:           sdk.MustNewDecFromStr("0.003016591251885370"),
			AskQuantity:        sdk.NewInt(200000000),
			BidQuantity:        sdk.NewInt(160000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.960000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("9.930000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("9.900000000000000000"),
			BidCount:           4,
			AskCount:           4,
			RemCount:           0,
			InvalidStatusCount: 0,
			TotalCount:         8,
		},
	} {
		suite.Run(tc.Name, func() {
			result := cmd.GetResult(tc.Orders, suite.params)
			suite.Require().EqualValues(result.MidPrice, tc.MidPrice)
			suite.Require().EqualValues(result.Spread, tc.Spread)
			suite.Require().EqualValues(result.AskWidth, tc.AskWidth)
			suite.Require().EqualValues(result.BidWidth, tc.BidWidth)
			suite.Require().EqualValues(result.AskQuantity, tc.AskQuantity)
			suite.Require().EqualValues(result.BidQuantity, tc.BidQuantity)
			suite.Require().EqualValues(result.AskMaxPrice, tc.AskMaxPrice)
			suite.Require().EqualValues(result.AskMinPrice, tc.AskMinPrice)
			suite.Require().EqualValues(result.BidMaxPrice, tc.BidMaxPrice)
			suite.Require().EqualValues(result.BidMinPrice, tc.BidMinPrice)
			suite.Require().EqualValues(result.BidCount, tc.BidCount)
			suite.Require().EqualValues(result.AskCount, tc.AskCount)
			suite.Require().EqualValues(result.RemCount, tc.RemCount)
			suite.Require().EqualValues(result.InvalidStatusCount, tc.InvalidStatusCount)
			suite.Require().EqualValues(result.TotalCount, tc.TotalCount)
			fmt.Println(result)
		})
	}
}
