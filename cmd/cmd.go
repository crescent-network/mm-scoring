package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ttypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"

	chain "github.com/crescent-network/crescent/v5/app"
	appparams "github.com/crescent-network/crescent/v5/app/params"
	crecmd "github.com/crescent-network/crescent/v5/cmd/crescentd/cmd"
	liquiditytypes "github.com/crescent-network/crescent/v5/x/liquidity/types"
	marketmakertypes "github.com/crescent-network/crescent/v5/x/marketmaker/types"
	minttypes "github.com/crescent-network/crescent/v5/x/mint/types"
)

type Context struct {
	StartHeight          int64
	LastHeight           int64
	LastCalcHeight       int64
	LastScoringHeight    int64
	LastScoringBlockTime *time.Time
	Month                int
	Hours                int
	LastHour             int

	LiquidityClient   liquiditytypes.QueryClient
	MarketMakerClient marketmakertypes.QueryClient
	MintClient        minttypes.QueryClient

	ParamsMap ParamsMap

	RpcWebsocketClient *rpchttp.HTTP
	WebsocketCtx       context.Context
	Enc                appparams.EncodingConfig

	mmMap map[uint64]map[string]*MM

	LastScoringBlock *Block

	Config
}

type MM struct {
	Address string
	PairId  uint64

	Eligible bool
	Apply    bool

	// No valid orders longer than MaxDowntime in a row
	SerialDownTime int
	// No valid orders longer than MaxTotalDowntime total in an hour
	TotalDownTime int

	DownThisHour bool

	// Total Live Hours
	TotalLiveHours int

	// Live Hours for the day
	LiveHours int

	// total live days for this month
	LiveDays int

	// TODO: update when end of month
	ThisMonthEligibility bool

	Score      sdk.Dec
	ScoreRatio sdk.Dec
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
	GrpcEndpoint: "13.124.45.5:9090",
	//GrpcEndpoint:   "127.0.0.1:9090",
	RpcEndpoint: "tcp://13.124.45.5:26657",
	//RpcEndpoint:    "tcp://127.0.0.1:26657",
	SimulationMode: true,
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
			ctx.mmMap = map[uint64]map[string]*MM{}
			ctx.LastScoringBlock = &Block{}

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
	fmt.Println("dialing: ", ctx.Config.GrpcEndpoint)
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

	// pair => height => addr => Result
	OrderMapByHeight := map[uint64]map[int64]map[string]*Result{}
	// pair => height => Summation C
	SumCMapByHeight := map[uint64]map[int64]sdk.Dec{}

	time.Sleep(1000000000)

	go Dashboard(ctx)

	// iterate from startHeight
	for i := ctx.StartHeight; ; {
		blockTime, err := QueryLastBlockTimeGRPC(ctx.MintClient, i)

		if err != nil || blockTime == nil || i > ctx.LastHeight {
			fmt.Println("waiting for the next block...")
			time.Sleep(1000000000)
			continue
		}

		// init time hours, months
		if ctx.StartHeight == i {
			ctx.Month = int(blockTime.Month())
			ctx.LastHour = blockTime.Hour()
		}

		fmt.Println("----------")
		fmt.Println(i, blockTime.String())

		// checking pruning height
		marketmakerparams, err := QueryMarketMakerParamsGRPC(ctx.MarketMakerClient, i)
		if err != nil {
			fmt.Println("pruning height", i)
			panic("not pruning data")
		}
		// TODO: checking change of params, start date

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
				ctx.mmMap[pair.PairId] = map[string]*MM{}
			}

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
					ctx.mmMap[pair.PairId][order.Orderer] = &MM{
						Address: order.Orderer,
						PairId:  pair.PairId,
					}
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
				OrderMapByHeight[pair.PairId][i][k] = result
				//// TODO: summation only eligible mm
				//if ctx.mmMap[pair.PairId][k].Eligible {
				//
				//}
				if !result.Live {
					ctx.mmMap[pair.PairId][k].SerialDownTime = ctx.mmMap[pair.PairId][k].SerialDownTime + 1
					ctx.mmMap[pair.PairId][k].TotalDownTime = ctx.mmMap[pair.PairId][k].TotalDownTime + 1
					fmt.Println("add downtime", pair.PairId, k, ctx.mmMap[pair.PairId][k].SerialDownTime, ctx.mmMap[pair.PairId][k].TotalDownTime)

					// Downtime checking
					if ctx.mmMap[pair.PairId][k].SerialDownTime > int(ctx.ParamsMap.Common.MaxDowntime) ||
						ctx.mmMap[pair.PairId][k].TotalDownTime > int(ctx.ParamsMap.Common.MaxTotalDowntime) {
						ctx.mmMap[pair.PairId][k].DownThisHour = true
					}
				}
				if ctx.LastHour != blockTime.Hour() && !ctx.mmMap[pair.PairId][k].DownThisHour {
					ctx.mmMap[pair.PairId][k].LiveHours += 1
					ctx.mmMap[pair.PairId][k].TotalLiveHours += 1
					if blockTime.Hour() == 0 {
						// LiveDay is added as LiveHour is equal or larger than MinHours in a day
						if ctx.mmMap[pair.PairId][k].LiveHours >= int(ctx.ParamsMap.Common.MinHours) {
							ctx.mmMap[pair.PairId][k].LiveDays += 1
						}
						ctx.mmMap[pair.PairId][k].LiveHours = 0
					}
				}

				SumCMapByHeight[pair.PairId][i] = SumCMapByHeight[pair.PairId][i].Add(OrderMapByHeight[pair.PairId][i][k].CMin)
			}
		}
		ctx.LastCalcHeight = i
		i++

		fmt.Println("StartHeight :", ctx.StartHeight)
		fmt.Println("LastHeight :", ctx.LastHeight)
		fmt.Println("LastScoringHeight :", ctx.LastScoringHeight)

		// next hour
		if ctx.LastHour != blockTime.Hour() {
			ctx.Hours += 1
			ctx.LastHour = blockTime.Hour()

			if ctx.Month != int(blockTime.Month()) {
				// TODO: reset scoring
				// TODO: checking LiveDay, MinDays
			}
		}

		// scoring every 1000 blocks
		if i%1000 == 0 {
			ctx.LastScoringHeight = i
			ctx.LastScoringBlockTime = blockTime
			ctx.LastScoringBlock.Height = i
			ctx.LastScoringBlock.Time = blockTime

			// checking eligible and applied
			mmResult, err := QueryMarketMakersGRPC(ctx.MarketMakerClient, i)
			if err != nil {
				panic(err)
			}
			for _, mm := range mmResult {
				if _, ok := ctx.mmMap[mm.PairId][mm.Address]; ok {
					if mm.Eligible {
						ctx.mmMap[mm.PairId][mm.Address].Eligible = true
					} else {
						ctx.mmMap[mm.PairId][mm.Address].Apply = true
					}
				}
			}

			for pair, mms := range ctx.mmMap {
				ScoreSum := sdk.ZeroDec()
				for _, mm := range mms {
					sumCQuoSumC := sdk.ZeroDec()
					for height, v := range OrderMapByHeight[pair] {
						// summation CQuoSumC
						if !SumCMapByHeight[pair][height].IsZero() && v[mm.Address] != nil {
							sumCQuoSumC = sumCQuoSumC.Add(v[mm.Address].CMin.QuoTruncate(SumCMapByHeight[pair][height]))
						}
					}
					U := sdk.NewDec(int64(mm.TotalLiveHours)).Quo(sdk.NewDec(int64(ctx.Hours)))
					U3 := U.Power(3)
					Score := sumCQuoSumC.Mul(U3)
					ScoreSum = ScoreSum.Add(Score)
					ctx.mmMap[pair][mm.Address].Score = Score

					fmt.Println(i, mm.Address, mm.PairId, U3, sumCQuoSumC.Mul(U3), sumCQuoSumC)
				}
				for _, mm := range mms {
					ctx.mmMap[pair][mm.Address].ScoreRatio = ctx.mmMap[pair][mm.Address].Score.QuoTruncate(ScoreSum)
				}
			}

			// TODO: temporary debugging output
			output(OrderMapByHeight, "output.json")
			output(SumCMapByHeight, "output-sum.json")
			output(ctx.mmMap, "output-mmMap.json")

		}
		// TODO: reset when next month
	}
	return nil
}

type Score struct {
	//Height    int64
	//BlockTime *time.Time
	Block *Block
	Score map[uint64]map[string]*MM
}

type Block struct {
	Height int64
	Time   *time.Time
}

func Dashboard(ctx Context) {
	r := gin.New()

	r.GET("/score", func(c *gin.Context) {
		c.JSON(http.StatusOK, Score{
			//Height:    ctx.LastScoringHeight,
			//BlockTime: ctx.LastScoringBlockTime,
			Block: ctx.LastScoringBlock,
			Score: ctx.mmMap,
		})
	})

	r.Run()
}
