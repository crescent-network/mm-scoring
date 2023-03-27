package cmd_test

import (
	"fmt"
	"testing"

	chain "github.com/crescent-network/crescent/v5/app"
	crecmd "github.com/crescent-network/crescent/v5/cmd/crescentd/cmd"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"

	utils "github.com/crescent-network/crescent/v5/types"
	"github.com/crescent-network/crescent/v5/x/liquidity/types"
	liquiditytypes "github.com/crescent-network/crescent/v5/x/liquidity/types"
	marketmakertypes "github.com/crescent-network/crescent/v5/x/marketmaker/types"

	"github.com/crescent-network/mm-scoring/cmd"
)

type OrderTestSuite struct {
	suite.Suite

	pm  cmd.ParamsMap
	ctx cmd.Context
}

func TestOrderTestSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

func (suite *OrderTestSuite) SetupSuite() {
	suite.pm = cmd.ParamsMap{
		Common: marketmakertypes.DefaultCommon,
		IncentivePairsMap: map[uint64]marketmakertypes.IncentivePair{
			1: {
				PairId:          1,
				UpdateTime:      utils.ParseTime("2022-09-01T00:00:00Z"),
				IncentiveWeight: sdk.MustNewDecFromStr("0.1"),
				MaxSpread:       sdk.MustNewDecFromStr("0.012"),
				MinWidth:        sdk.MustNewDecFromStr("0.002"),
				MinDepth:        sdk.NewInt(100000000),
			},
		},
	}

	crecmd.GetConfig()
	suite.ctx.Config = cmd.DefaultConfig
	suite.ctx.Enc = chain.MakeEncodingConfig()

	// ===================================== Create a connection to the gRPC server ====================================
	grpcConn, err := grpc.Dial(
		suite.ctx.Config.GrpcEndpoint, // your gRPC server address.
		grpc.WithInsecure(),           // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		//grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		panic(err)
	}
	//defer grpcConn.Close()

	suite.ctx.LiquidityClient = liquiditytypes.NewQueryClient(grpcConn)
}

func (suite *OrderTestSuite) TestGetResult() {
	suite.pm = cmd.ParamsMap{
		//Common: marketmakertypes.DefaultCommon,
		Common: marketmakertypes.Common{
			MinOpenRatio:      sdk.MustNewDecFromStr("0.5"),
			MinOpenDepthRatio: sdk.MustNewDecFromStr("0.1"),
			MaxDowntime:       uint32(20),
			MaxTotalDowntime:  uint32(100),
			MinHours:          uint32(16),
			MinDays:           uint32(22),
		},
		IncentivePairsMap: map[uint64]marketmakertypes.IncentivePair{
			1: {
				PairId:          1,
				UpdateTime:      utils.ParseTime("2022-09-01T00:00:00Z"),
				IncentiveWeight: sdk.MustNewDecFromStr("0.1"),
				MaxSpread:       sdk.MustNewDecFromStr("0.012"),
				MinWidth:        sdk.MustNewDecFromStr("0.002"),
				MinDepth:        sdk.NewInt(100000000),
			},
		},
	}

	for _, tc := range []struct {
		Name               string
		Orders             []types.Order
		MidPrice           sdk.Dec
		Spread             sdk.Dec
		AskWidth           sdk.Dec
		BidWidth           sdk.Dec
		AskDepth           sdk.Int
		BidDepth           sdk.Int
		AskMaxPrice        sdk.Dec
		AskMinPrice        sdk.Dec
		BidMaxPrice        sdk.Dec
		BidMinPrice        sdk.Dec
		BidCount           int
		AskCount           int
		RemCount           int
		InvalidStatusCount int
		TotalCount         int
		Live               bool
		CBid               sdk.Dec
		CAsk               sdk.Dec
		CMin               sdk.Dec
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
			MidPrice:           sdk.MustNewDecFromStr("1.175000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.042553191489361702"),
			AskWidth:           sdk.MustNewDecFromStr("0.085106382978723404"),
			BidWidth:           sdk.MustNewDecFromStr("0.127659574468085106"),
			AskDepth:           sdk.NewInt(20000000000),
			BidDepth:           sdk.NewInt(59000000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("1.300000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("1.200000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("1.150000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("1.000000000000000000"),
			BidCount:           4,
			AskCount:           2,
			RemCount:           0,
			InvalidStatusCount: 2,
			TotalCount:         8,
			Live:               false, // spread over
			CAsk:               sdk.MustNewDecFromStr("0.0"),
			CBid:               sdk.MustNewDecFromStr("0.0"),
			CMin:               sdk.MustNewDecFromStr("0.0"),
		},
		{
			Name: "case block 1 a - spec example",
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
			Spread:             sdk.MustNewDecFromStr("0.003016591251885369"),
			AskWidth:           sdk.MustNewDecFromStr("0.003016591251885369"),
			BidWidth:           sdk.MustNewDecFromStr("0.003016591251885369"),
			AskDepth:           sdk.NewInt(200000000),
			BidDepth:           sdk.NewInt(160000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.960000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("9.930000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("9.900000000000000000"),
			BidCount:           4,
			AskCount:           4,
			RemCount:           0,
			InvalidStatusCount: 0,
			TotalCount:         8,
			Live:               true,
			CBid:               sdk.MustNewDecFromStr("29095680130610.547241037833592663"),
			CAsk:               sdk.MustNewDecFromStr("36369600163263.184051297291990829"),
			CMin:               sdk.MustNewDecFromStr("29095680130610.547241037833592663"),
		},
		{
			Name: "case block 2, a - spec example",
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
					OpenAmount: sdk.NewInt(40000000),
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
					OpenAmount: sdk.NewInt(0),
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
					OpenAmount: sdk.NewInt(5000000),
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
			MidPrice:           sdk.MustNewDecFromStr("9.935000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.005032712632108706"),
			AskWidth:           sdk.MustNewDecFromStr("0.003019627579265223"),
			BidWidth:           sdk.MustNewDecFromStr("0.001006542526421741"),
			AskDepth:           sdk.NewInt(190000000),
			BidDepth:           sdk.NewInt(80000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.960000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("9.910000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("9.900000000000000000"),
			BidCount:           2,
			AskCount:           4,
			RemCount:           2,
			InvalidStatusCount: 0,
			TotalCount:         8,
			Live:               false, // MinWidth
			CBid:               sdk.MustNewDecFromStr("0.0"),
			CAsk:               sdk.MustNewDecFromStr("0.0"),
			CMin:               sdk.MustNewDecFromStr("0.0"),
		},
		{
			Name: "case block 1 b - spec example",
			Orders: []types.Order{
				{
					Id:         478556,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.990000000000000000"),
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(80000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(80000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(80000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
			},
			MidPrice:           sdk.MustNewDecFromStr("9.945000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.005027652086475615"),
			AskWidth:           sdk.MustNewDecFromStr("0.002011060834590246"),
			BidWidth:           sdk.MustNewDecFromStr("0.002011060834590246"),
			AskDepth:           sdk.NewInt(225000000),
			BidDepth:           sdk.NewInt(240000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.970000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("9.920000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("9.900000000000000000"),
			BidCount:           3,
			AskCount:           3,
			RemCount:           0,
			InvalidStatusCount: 0,
			TotalCount:         6,
			Live:               true,
			CBid:               sdk.MustNewDecFromStr("23025840261223.641239365107000888"),
			CAsk:               sdk.MustNewDecFromStr("21586725244897.163661904787813332"),
			CMin:               sdk.MustNewDecFromStr("21586725244897.163661904787813332"),
		},
		{
			Name: "case block 2 b - spec example",
			Orders: []types.Order{
				{
					Id:         478556,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.990000000000000000"),
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(20000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(80000000),
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
					Amount:     sdk.NewInt(80000000),
					OpenAmount: sdk.NewInt(80000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
			},
			MidPrice:           sdk.MustNewDecFromStr("9.945000000000000000"),
			Spread:             sdk.MustNewDecFromStr("0.005027652086475615"),
			AskWidth:           sdk.MustNewDecFromStr("0.002011060834590246"),
			BidWidth:           sdk.MustNewDecFromStr("0.002011060834590246"),
			AskDepth:           sdk.NewInt(225000000),
			BidDepth:           sdk.NewInt(180000000),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.970000000000000000"),
			BidMaxPrice:        sdk.MustNewDecFromStr("9.920000000000000000"),
			BidMinPrice:        sdk.MustNewDecFromStr("9.900000000000000000"),
			BidCount:           3,
			AskCount:           3,
			RemCount:           0,
			InvalidStatusCount: 0,
			TotalCount:         6,
			Live:               true,
			CBid:               sdk.MustNewDecFromStr("13531149861224.161920970678268334"),
			CAsk:               sdk.MustNewDecFromStr("21586725244897.163661904787813332"),
			CMin:               sdk.MustNewDecFromStr("13531149861224.161920970678268334"),
		},
		{
			Name: "1-side order",
			Orders: []types.Order{
				{
					Id:         478556,
					PairId:     1,
					MsgHeight:  478556,
					Orderer:    "cre18s7e4ag2stm85jwlvy7fs7hufx8xc0sg3efwuy",
					Direction:  types.OrderDirectionSell, // 2
					Price:      sdk.MustNewDecFromStr("9.990000000000000000"),
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
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
					Amount:     sdk.NewInt(75000000),
					OpenAmount: sdk.NewInt(75000000),
					BatchId:    399117,
					Status:     types.OrderStatusNotMatched, // 2
				},
			},
			MidPrice:           sdk.ZeroDec(),
			Spread:             sdk.ZeroDec(),
			AskWidth:           sdk.ZeroDec(),
			BidWidth:           sdk.ZeroDec(),
			AskDepth:           sdk.NewInt(225000000),
			BidDepth:           sdk.NewInt(0),
			AskMaxPrice:        sdk.MustNewDecFromStr("9.990000000000000000"),
			AskMinPrice:        sdk.MustNewDecFromStr("9.970000000000000000"),
			BidMaxPrice:        sdk.ZeroDec(),
			BidMinPrice:        sdk.ZeroDec(),
			BidCount:           0,
			AskCount:           3,
			RemCount:           0,
			InvalidStatusCount: 0,
			TotalCount:         3,
			Live:               false,
			CBid:               sdk.ZeroDec(),
			CAsk:               sdk.ZeroDec(),
			CMin:               sdk.ZeroDec(),
		},
	} {
		suite.Run(tc.Name, func() {
			result := cmd.NewResult()
			result.Orders = tc.Orders
			result = cmd.SetResult(result, suite.pm, tc.Orders[0].PairId)
			suite.Require().EqualValues(result.MidPrice, tc.MidPrice)
			suite.Require().EqualValues(result.Spread, tc.Spread)
			suite.Require().EqualValues(result.AskWidth, tc.AskWidth)
			suite.Require().EqualValues(result.BidWidth, tc.BidWidth)
			suite.Require().EqualValues(result.AskDepth, tc.AskDepth)
			suite.Require().EqualValues(result.BidDepth, tc.BidDepth)
			suite.Require().EqualValues(result.AskMaxPrice, tc.AskMaxPrice)
			suite.Require().EqualValues(result.AskMinPrice, tc.AskMinPrice)
			suite.Require().EqualValues(result.BidMaxPrice, tc.BidMaxPrice)
			suite.Require().EqualValues(result.BidMinPrice, tc.BidMinPrice)
			suite.Require().EqualValues(result.BidCount, tc.BidCount)
			suite.Require().EqualValues(result.AskCount, tc.AskCount)
			suite.Require().EqualValues(result.RemCount, tc.RemCount)
			suite.Require().EqualValues(result.InvalidStatusCount, tc.InvalidStatusCount)
			suite.Require().EqualValues(result.TotalCount, tc.TotalCount)
			suite.Require().EqualValues(result.Live, tc.Live)
			suite.Require().EqualValues(result.CBid, tc.CBid)
			suite.Require().EqualValues(result.CAsk, tc.CAsk)
			suite.Require().EqualValues(result.CMin, tc.CMin)
			fmt.Println(result.Live, result.CMin, result.CBid, result.CAsk)
			fmt.Println(result)
		})
	}
}

func (suite *OrderTestSuite) TestMMOrder() {
	height := int64(1987252)
	blockTime := utils.ParseTime("2022-09-15T09:07:04.561862Z")

	params := liquiditytypes.DefaultParams()

	lastPrice := sdk.MustNewDecFromStr("1.111200000000000000")
	pair := liquiditytypes.Pair{
		Id:             1,
		BaseCoinDenom:  "ubcre",
		QuoteCoinDenom: "ucre",
		EscrowAddress:  "cre17u9nx0h9cmhypp6cg9lf4q8ku9l3k8mz232su7m28m39lkz25dgqw9sanj",
		LastOrderId:    1000,
		LastPrice:      &lastPrice,
		CurrentBatchId: 1000,
	}

	msg := cmd.MsgMMOrder{
		Orderer:       "cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k",
		PairId:        1,
		MaxSellPrice:  sdk.MustNewDecFromStr("1.21"),
		MinSellPrice:  sdk.MustNewDecFromStr("1.115"),
		SellAmount:    sdk.NewInt(500000),
		MaxBuyPrice:   sdk.MustNewDecFromStr("1.1"),
		MinBuyPrice:   sdk.MustNewDecFromStr("1.05"),
		BuyAmount:     sdk.NewInt(500000),
		OrderLifespan: 0,
	}
	orders, err := cmd.MMOrder(pair, height, &blockTime, &params, &msg)
	suite.Require().NoError(err)

	fmt.Println(orders)
}
