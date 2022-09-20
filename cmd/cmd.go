package cmd

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	utils "github.com/crescent-network/crescent/v3/types"
	"github.com/crescent-network/crescent/v3/x/liquidity/amm"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ttypes "github.com/tendermint/tendermint/types"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"

	chain "github.com/crescent-network/crescent/v3/app"
	appparams "github.com/crescent-network/crescent/v3/app/params"
	crecmd "github.com/crescent-network/crescent/v3/cmd/crescentd/cmd"
	liquiditytypes "github.com/crescent-network/crescent/v3/x/liquidity/types"
	marketmakertypes "github.com/crescent-network/crescent/v3/x/marketmaker/types"
	minttypes "github.com/crescent-network/crescent/v3/x/mint/types"
)

type Context struct {
	StartHeight       int64
	LastHeight        int64
	LastScoringHeight int64

	LiquidityClient   liquiditytypes.QueryClient
	MarketMakerClient marketmakertypes.QueryClient
	MintClient        minttypes.QueryClient

	ParamsMap ParamsMap

	RpcWebsocketClient *rpchttp.HTTP
	WebsocketCtx       context.Context
	Enc                appparams.EncodingConfig

	// TODO: eligible, apply mm map
	mmMap         map[uint64]map[string]struct{}
	mmMapEligible map[uint64]map[string]struct{}

	Config
}

type Config struct {
	GrpcEndpoint string
	RpcEndpoint  string

	// when if true, generate mock orders
	SimulationMode bool
}

type ParamsMap struct {
	Common            marketmakertypes.Common
	IncentivePairsMap map[uint64]marketmakertypes.IncentivePair
}

var DefaultConfig = Config{
	GrpcEndpoint:   "127.0.0.1:9090",
	RpcEndpoint:    "tcp://127.0.0.1:26657",
	SimulationMode: false,
}

func NewScoringCmd() *cobra.Command {
	// set prefix to cre from cosmos
	crecmd.GetConfig()
	cmd := &cobra.Command{
		Use:  "mm-scoring [start-height]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ctx Context
			ctx.Config = DefaultConfig
			ctx.mmMap = map[uint64]map[string]struct{}{}

			startHeight, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("parse start-Height: %w", err)
			}

			ctx.StartHeight = startHeight

			grpcEndpoint, _ := cmd.Flags().GetString("grpc")
			if grpcEndpoint != ctx.Config.GrpcEndpoint {
				ctx.Config.GrpcEndpoint = grpcEndpoint
			}

			rpcEndpoint, _ := cmd.Flags().GetString("rpc")
			if rpcEndpoint != ctx.Config.RpcEndpoint {
				ctx.Config.RpcEndpoint = rpcEndpoint
			}

			simStr, _ := cmd.Flags().GetString("sim")
			if simStr != "" {
				sim, err := strconv.ParseBool(simStr)
				if err != nil {
					return fmt.Errorf("parse disabled flag: %w", err)
				}
				ctx.Config.SimulationMode = sim
			}

			return Main(ctx)
		},
	}
	cmd.Flags().String("grpc", DefaultConfig.GrpcEndpoint, "set grpc endpoint")
	cmd.Flags().String("rpc", DefaultConfig.RpcEndpoint, "set rpc endpoint")
	cmd.Flags().String("sim", "false", "set rpc endpoint")
	return cmd
}

func Main(ctx Context) error {
	ctx.Enc = chain.MakeEncodingConfig()

	// ===================================== Create a connection to the gRPC server ====================================
	grpcConn, err := grpc.Dial(
		ctx.Config.GrpcEndpoint, // your gRPC server address.
		grpc.WithInsecure(),     // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		//grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	ctx.LiquidityClient = liquiditytypes.NewQueryClient(grpcConn)
	ctx.MarketMakerClient = marketmakertypes.NewQueryClient(grpcConn)
	ctx.MintClient = minttypes.NewQueryClient(grpcConn)
	// =================================================================================================================

	// ===================================== websocket, update ctx.LastHeight ==========================================
	wc, err := rpchttp.New(ctx.Config.RpcEndpoint, "/websocket")
	if err != nil {
		log.Fatal(err)
	}
	ctx.RpcWebsocketClient = wc

	err = ctx.RpcWebsocketClient.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.RpcWebsocketClient.Stop()
	wcCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx.WebsocketCtx = wcCtx

	query := "tm.event = 'NewBlock'"
	res, err := ctx.RpcWebsocketClient.Subscribe(ctx.WebsocketCtx, "test-wc", query)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for e := range res {
			switch data := e.Data.(type) {
			case ttypes.EventDataNewBlock:
				fmt.Printf("New Block: %d \n", data.Block.Height)
				ctx.LastHeight = data.Block.Height
				if err != nil {
					fmt.Println(err)
				}
				break
			}
		}
	}()

	// =================================================================================================================

	fmt.Println("version :", "v0.1.1")
	fmt.Println("Input Start Height :", ctx.StartHeight)
	fmt.Println("Simulation mode :", ctx.Config.SimulationMode)

	startTime := time.Now()

	// TODO: add result(C) for each mm on each height
	// pair => height => Address => []order, TODO: add result(c)
	OrderMapByHeight := map[uint64]map[int64]map[string]*Result{}
	// pair => height => Summation C
	SumCMapByHeight := map[uint64]map[int64]sdk.Dec{}

	//CMapByHeight := map[uint64]map[int64]map[string]Result{}
	//OrderMapByHeight := map[uint64]map[int64]map[string][]liquiditytypes.Order{}
	//CMapByHeight := map[uint64]map[int64]map[string]Result{}

	// pair => height
	//SummationCMap := map[uint64]map[int64]sdk.Dec{}

	// pair => height => address => result
	//ResultMapByHeight := map[uint64]map[int64]map[string]Result{}

	// TODO: GET params, considering update height
	//app.MarketMakerKeeper.GetParams(ctx)

	time.Sleep(1000000000)

	// iterate from startHeight
	// TODO: if fail, retry new height with delay or websocket chan
	for i := ctx.StartHeight; ; {
		blockTime, err := QueryLastBlockTimeGRPC(ctx.MintClient, i)

		if err != nil || blockTime == nil || i > ctx.LastHeight {
			fmt.Println("waiting for the next block...")
			time.Sleep(1000000000)
			continue
		}

		fmt.Println("----------")
		fmt.Println(i, blockTime.String())
		// TODO: uptime map
		TimeToHour(blockTime)

		// checking pruning height
		marketmakerparams, err := QueryMarketMakerParamsGRPC(ctx.MarketMakerClient, i)
		if err != nil {
			fmt.Println("pruning height", i)
			panic("not pruning data")
		}

		// need when only using generator
		liquidityparams, err := QueryLiquidityParamsGRPC(ctx.LiquidityClient, i)
		if err != nil {
			panic(err)
		}

		ctx.ParamsMap = GetParamsMap(marketmakerparams.Params, blockTime)

		pairs, err := QueryPairsGRPC(ctx.LiquidityClient, i)
		if err != nil {
			panic(err)
		}
		pairsMap := GetPairsMap(pairs)

		for _, pair := range ctx.ParamsMap.IncentivePairsMap {
			// TODO: continue or use last state when updateTime is future

			if _, ok := ctx.mmMap[pair.PairId]; !ok {
				ctx.mmMap[pair.PairId] = map[string]struct{}{}
			}

			// TODO: replacing generator
			var orders []liquiditytypes.Order
			if ctx.Config.SimulationMode {
				orders, err = GenerateMockOrders(pairsMap[pair.PairId], ctx.ParamsMap.IncentivePairsMap[pair.PairId], i, blockTime, liquidityparams, ctx.mmMap[pair.PairId])
				if err != nil {
					return err
				}
			} else {
				orders, err = QueryOrdersGRPC(ctx.LiquidityClient, pair.PairId, i)
				if err != nil {
					return err
				}
			}
			for _, order := range orders {
				// scoring only mm order type
				if order.Type != liquiditytypes.OrderTypeMM {
					continue
				}

				if _, ok := ctx.mmMap[pair.PairId][order.Orderer]; !ok {
					ctx.mmMap[pair.PairId][order.Orderer] = struct{}{}
				}

				if _, ok := OrderMapByHeight[pair.PairId]; !ok {
					OrderMapByHeight[pair.PairId] = map[int64]map[string]*Result{}
				}
				if _, ok := OrderMapByHeight[pair.PairId][i]; !ok {
					OrderMapByHeight[pair.PairId][i] = map[string]*Result{}
				}

				if OrderMapByHeight[pair.PairId][i][order.Orderer] == nil {
					OrderMapByHeight[pair.PairId][i][order.Orderer] = NewResult()
				}
				OrderMapByHeight[pair.PairId][i][order.Orderer].Orders = append(OrderMapByHeight[pair.PairId][i][order.Orderer].Orders, order)
			}
			// init SumCMapByHeight for each pair, height
			if _, ok := SumCMapByHeight[pair.PairId]; !ok {
				SumCMapByHeight[pair.PairId] = map[int64]sdk.Dec{}
			}
			if _, ok := SumCMapByHeight[pair.PairId][i]; !ok {
				SumCMapByHeight[pair.PairId][i] = sdk.ZeroDec()
			}
			// calc C and summation min C per each addresses
			for k, v := range OrderMapByHeight[pair.PairId][i] {
				result := SetResult(v, ctx.ParamsMap, pair.PairId)
				if result == nil {
					continue
				}
				// TODO: summation only eligible mm
				OrderMapByHeight[pair.PairId][i][k] = result
				SumCMapByHeight[pair.PairId][i] = SumCMapByHeight[pair.PairId][i].Add(OrderMapByHeight[pair.PairId][i][k].CMin)
			}
		}
		ctx.LastScoringHeight = i
		i++

		fmt.Println("StartHeight :", ctx.StartHeight)
		fmt.Println("LastHeight :", ctx.LastHeight)
		fmt.Println("LastScoringHeight :", ctx.LastScoringHeight)
		output(OrderMapByHeight, "output.json")
		output(SumCMapByHeight, "output-sum.json")
		output(ctx.mmMap, "output-mmMap.json")

		//block = blockStore.LoadBlock(i)
		if i%100 == 0 {
			fmt.Println(i, time.Now().Sub(startTime))
			// TODO: calc tmp score with summation and uptime calc
			for pair, mms := range ctx.mmMap {
				for mm, _ := range mms {
					sumCQuoSumC := sdk.ZeroDec()
					for height, v := range OrderMapByHeight[pair] {
						// TODO: fix div by zero
						// summation CQuoSumC
						if !SumCMapByHeight[pair][height].IsZero() && v[mm] != nil {
							sumCQuoSumC = sumCQuoSumC.Add(v[mm].CMin.QuoTruncate(SumCMapByHeight[pair][height]))
						}
					}
					// TODO: make map, uptime^3
					//fmt.Println(i, pair, mm, sumCQuoSumC)
				}
			}
		}

		// TODO: 20 serial block Obligation
		// TODO: 100 total block Obligation
		// TODO: Hour Obligation
		// TODO: Daily Obligation
		// TODO: Monthly Obligation

		// TODO: reset when next month
		// TODO: not eligible mm -> only check obligation

	}
	return nil
}

// TODO: make generator
func GenerateMockOrders(pair liquiditytypes.Pair, incentivePair marketmakertypes.IncentivePair, height int64, blockTime *time.Time, params *liquiditytypes.Params, mmMap map[string]struct{}) (orders []liquiditytypes.Order, err error) {
	r := rand.New(rand.NewSource(0))

	// TODO: add mm address
	mmMap["cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk"] = struct{}{}
	mmMap["cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev"] = struct{}{}

	tickPrec := int(params.TickPrecision)

	// TODO: rand seed
	for mm, _ := range mmMap {
		orderOrNot := rand.Intn(2)
		if orderOrNot == 1 {
			continue
		}
		//amountOverOrNot := rand.Bool()
		//priceRangeValidOrNot := rand.Bool()

		sellAmount := incentivePair.MinDepth.MulRaw(int64(params.MaxNumMarketMakingOrderTicks)).ToDec().Mul(
			utils.RandomDec(r, utils.ParseDec("0.95"), utils.ParseDec("1.45"))).TruncateInt()

		buyAmount := incentivePair.MinDepth.MulRaw(int64(params.MaxNumMarketMakingOrderTicks)).ToDec().Mul(
			utils.RandomDec(r, utils.ParseDec("0.95"), utils.ParseDec("1.45"))).TruncateInt()

		simtypes.RandomDecAmount(r, sdk.NewDecWithPrec(1, 2))

		msg := liquiditytypes.MsgMMOrder{
			Orderer:       mm,
			PairId:        1,
			MaxSellPrice:  amm.PriceToDownTick(pair.LastPrice.Add(pair.LastPrice.Mul(utils.RandomDec(r, utils.ParseDec("0.011"), utils.ParseDec("0.016")))), tickPrec),
			MinSellPrice:  amm.PriceToUpTick(pair.LastPrice.Add(pair.LastPrice.Mul(utils.RandomDec(r, utils.ParseDec("0.001"), utils.ParseDec("0.01")))), tickPrec),
			SellAmount:    sellAmount,
			MaxBuyPrice:   amm.PriceToUpTick(pair.LastPrice.Sub(pair.LastPrice.Mul(utils.RandomDec(r, utils.ParseDec("0.001"), utils.ParseDec("0.01")))), tickPrec),
			MinBuyPrice:   amm.PriceToDownTick(pair.LastPrice.Sub(pair.LastPrice.Mul(utils.RandomDec(r, utils.ParseDec("0.011"), utils.ParseDec("0.016")))), tickPrec),
			BuyAmount:     buyAmount,
			OrderLifespan: 0,
		}

		newOrders, err := MMOrder(pair, height, blockTime, params, &msg)
		if err != nil {
			panic(err)
		} else {
			orders = append(orders, newOrders...)
		}
	}
	return orders, nil
}

func QueryOrdersGRPC(cli liquiditytypes.QueryClient, pairId uint64, height int64) (orderRes []liquiditytypes.Order, err error) {
	req := liquiditytypes.QueryOrdersRequest{
		PairId: pairId,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	var header metadata.MD

	res, err := cli.Orders(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return res.Orders, nil
}

func QueryMarketMakerParamsGRPC(cli marketmakertypes.QueryClient, height int64) (paramRes *marketmakertypes.QueryParamsResponse, err error) {
	req := marketmakertypes.QueryParamsRequest{}

	var header metadata.MD

	res, err := cli.Params(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func QueryLastBlockTimeGRPC(cli minttypes.QueryClient, height int64) (blockTime *time.Time, err error) {
	req := minttypes.QueryLastBlockTimeRequest{}

	var header metadata.MD

	res, err := cli.LastBlockTime(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return res.LastBlockTime, nil
}

func QueryPoolsGRPC(cli liquiditytypes.QueryClient, pairId uint64, height int64) (poolsRes []liquiditytypes.PoolResponse, err error) {
	req := liquiditytypes.QueryPoolsRequest{
		PairId:   pairId,
		Disabled: "false",
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	}

	var header metadata.MD

	res, err := cli.Pools(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return res.Pools, nil
}

func QueryLiquidityParamsGRPC(cli liquiditytypes.QueryClient, height int64) (paramRes *liquiditytypes.Params, err error) {
	req := liquiditytypes.QueryParamsRequest{}

	var header metadata.MD

	res, err := cli.Params(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return &res.Params, nil
}

func QueryPairsGRPC(cli liquiditytypes.QueryClient, height int64) (pairsRes []liquiditytypes.Pair, err error) {
	pairsReq := liquiditytypes.QueryPairsRequest{
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	}

	var header metadata.MD

	pairs, err := cli.Pairs(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&pairsReq,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return pairs.Pairs, nil
}

func QueryPairGRPC(cli liquiditytypes.QueryClient, pairId uint64, height int64) (pair liquiditytypes.Pair, err error) {
	pairsReq := liquiditytypes.QueryPairRequest{
		PairId: pairId,
	}

	var header metadata.MD

	pairRes, err := cli.Pair(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&pairsReq,
		grpc.Header(&header),
	)
	if err != nil {
		return pair, err
	}

	return pairRes.Pair, nil
}

func GetParamsMap(params marketmakertypes.Params, blockTime *time.Time) (pm ParamsMap) {
	pm.Common = params.Common
	// TODO: temporary generate mock pairs
	params.IncentivePairs = append(params.IncentivePairs, marketmakertypes.IncentivePair{
		PairId:          8,
		UpdateTime:      utils.ParseTime("2022-09-01T00:00:00Z"),
		IncentiveWeight: sdk.MustNewDecFromStr("0.9"),
		MaxSpread:       sdk.MustNewDecFromStr("0.006"),
		MinWidth:        sdk.MustNewDecFromStr("0.001"),
		MinDepth:        sdk.NewInt(600000000000000000),
	})

	// handle incentive pair's update time
	iMap := make(map[uint64]marketmakertypes.IncentivePair)
	for _, pair := range params.IncentivePairs {
		if pair.UpdateTime.Before(*blockTime) {
			iMap[pair.PairId] = pair
		}
	}

	pm.IncentivePairsMap = iMap
	//pm.IncentivePairsMap = params.IncentivePairsMap()

	return
}

func GetPairsMap(pairs []liquiditytypes.Pair) (pairsMap map[uint64]liquiditytypes.Pair) {
	pairsMap = map[uint64]liquiditytypes.Pair{}
	for _, pair := range pairs {
		pairsMap[pair.Id] = pair
	}
	return
}
