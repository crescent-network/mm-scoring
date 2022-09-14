package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ttypes "github.com/tendermint/tendermint/types"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"

	chain "github.com/crescent-network/crescent/v3/app"
	liquidityparams "github.com/crescent-network/crescent/v3/app/params"
	crecmd "github.com/crescent-network/crescent/v3/cmd/crescentd/cmd"
	liquiditytypes "github.com/crescent-network/crescent/v3/x/liquidity/types"
)

type Context struct {
	StartHeight       int64
	LastHeight        int64
	LastScoringHeight int64
	SyncStatus        bool // TODO: delete?

	LiquidityClient liquiditytypes.QueryClient
	// TODO: bank, etc

	RpcWebsocketClient *rpchttp.HTTP
	WebsocketCtx       context.Context

	Enc liquidityparams.EncodingConfig

	AccList  []string
	PairList []uint64 // if empty, all pairs
	Config
}

type Config struct {
	StartHeight        int64
	OrderbookKeepBlock int // zero == keep all?
	AllOrdersKeepBlock int
	PairPoolKeepBlock  int
	BalanceKeepBlock   int
	GrpcEndpoint       string
	RpcEndpoint        string
	//Dir                string
	// TODO: orderbook argument
}

var config = Config{
	GrpcEndpoint: "127.0.0.1:9090",
	RpcEndpoint:  "tcp://127.0.0.1:26657",
}

func NewScoringCmd() *cobra.Command {
	crecmd.GetConfig()
	cmd := &cobra.Command{
		// TODO: grpc endpoint
		Use:  "mm-scoring [start-height]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ctx Context
			ctx.Config = config

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

			return Main(ctx)
		},
	}
	cmd.Flags().String("grpc", "127.0.0.1:9090", "set grpc endpoint")
	cmd.Flags().String("rpc", "tcp://127.0.0.1:26657", "set rpc endpoint")
	return cmd
}

func Main(ctx Context) error {
	ctx.Enc = chain.MakeEncodingConfig()

	// ================================================= Create a connection to the gRPC server.
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
	// =========================================================================================

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
	txs, err := ctx.RpcWebsocketClient.Subscribe(ctx.WebsocketCtx, "test-wc", query)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for e := range txs {
			switch data := e.Data.(type) {
			case ttypes.EventDataNewBlock:
				fmt.Printf("Block %s - Height: %d \n", hex.EncodeToString(data.Block.Hash()), data.Block.Height)
				ctx.LastHeight = data.Block.Height
				if err != nil {
					fmt.Println(err)
				}
				break
			}
		}
	}()

	// ======================================== websocket end ==========================================================

	fmt.Println("version :", "v0.1.1")
	fmt.Println("Input Start Height :", ctx.StartHeight)

	startTime := time.Now()
	orderCount := 0
	orderTxCount := 0

	// Height => PairId => Order
	orderMap := map[int64]map[uint64][]liquiditytypes.Order{}
	// pair => Address => height => orders
	// TODO: // pair => height => Address => orders, delete this
	OrderMapByAddr := map[uint64]map[string]map[int64][]liquiditytypes.Order{}
	// TODO: add result(C) for each mm on each height
	// pair => height => Address => []order, TODO: add result(c)
	OrderMapByHeight := map[uint64]map[int64]map[string][]liquiditytypes.Order{}

	// pair => height
	//SummationCMap := map[uint64]map[int64]sdk.Dec{}

	// pair => height => address => result
	//ResultMapByHeight := map[uint64]map[int64]map[string]Result{}

	// TODO: convert to mmOrder and indexing by mm address
	mmOrderMap := map[int64]map[uint64][]liquiditytypes.MsgLimitOrder{}
	//mmOrderCancelMap := map[int64]map[uint64]liquiditytypes.MsgLimitOrder{}

	// TODO: OrderData list
	//OrderDataList := []OrderData{}

	// TODO: GET params, considering update height
	//app.MarketMakerKeeper.GetParams(ctx)

	// iterate from startHeight
	// TODO: if fail, retry new height with delay or websocket chan
	for i := ctx.StartHeight; ; {
		if i > ctx.LastHeight {
			time.Sleep(100000000)
			continue
		}

		// checking pruning height
		_, err := QueryParamsGRPC(ctx.LiquidityClient, i)
		if err != nil {
			fmt.Println("pruning height", i)
			time.Sleep(100000000)
			continue
		}

		//block = blockStore.LoadBlock(i)
		if i%10000 == 0 {
			fmt.Println(i, time.Now().Sub(startTime), orderCount, orderTxCount)
		}
		orderMap[i] = map[uint64][]liquiditytypes.Order{}
		mmOrderMap[i] = map[uint64][]liquiditytypes.MsgLimitOrder{}
		// Address -> pair -> height -> orders

		// Query paris
		pairs, err := QueryPairsGRPC(ctx.LiquidityClient, i)
		if err != nil {
			return err
		}
		for _, pair := range pairs {
			// TODO: check pruning by query params

			orders, err := QueryOrdersGRPC(ctx.LiquidityClient, pair.Id, i)
			if err != nil {
				return err
			}
			for _, order := range orders {
				// scoring only mm order type
				if order.Type != liquiditytypes.OrderTypeMM {
					continue
				}
				fmt.Println(pair.Id, order.Id)

				//pools, err := QueryPoolsGRPC(ctx.LiquidityClient, order.PairId, i)
				//if err != nil {
				//	return err
				//}
				//OrderDataList = append(OrderDataList, OrderData{
				//	Order:  order,
				//	Pools:  pools, // TODO: ??
				//	Height: i,
				//	//BlockTime: block.Time,
				//})

				orderCount++
				orderMap[i][pair.Id] = append(orderMap[i][pair.Id], order)

				// indexing order.PairId, address
				// TODO: filtering only mm address, mm order
				if _, ok := OrderMapByAddr[pair.Id]; !ok {
					OrderMapByAddr[pair.Id] = map[string]map[int64][]liquiditytypes.Order{}
				}
				if _, ok := OrderMapByAddr[pair.Id][order.Orderer]; !ok {
					OrderMapByAddr[pair.Id][order.Orderer] = map[int64][]liquiditytypes.Order{}
				}
				OrderMapByAddr[pair.Id][order.Orderer][i] = append(OrderMapByAddr[pair.Id][order.Orderer][i], order)

				// TODO: WIP OrderMapByHeight
				if _, ok := OrderMapByHeight[pair.Id]; !ok {
					OrderMapByHeight[pair.Id] = map[int64]map[string][]liquiditytypes.Order{}
				}
				if _, ok := OrderMapByHeight[pair.Id][i]; !ok {
					OrderMapByHeight[pair.Id][i] = map[string][]liquiditytypes.Order{}
				}
				// TODO: sorting order by price, buy, sell
				OrderMapByHeight[pair.Id][i][order.Orderer] = append(OrderMapByHeight[pair.Id][i][order.Orderer], order)
			}
			// TODO: GetResult for ResultMapByHeight, SummationCMap
		}
		ctx.LastScoringHeight = i
		i++

		fmt.Println("StartHeight :", ctx.StartHeight)
		fmt.Println("LastHeight :", ctx.LastHeight)
		fmt.Println("LastScoringHeight :", ctx.LastScoringHeight)
	}
	return nil
}

// TODO: query params for checking pruning

func QueryOrdersGRPC(cli liquiditytypes.QueryClient, pairId uint64, height int64) (poolsRes []liquiditytypes.Order, err error) {
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

// TODO: use to check pruning

func QueryParamsGRPC(cli liquiditytypes.QueryClient, height int64) (resParams *liquiditytypes.QueryParamsResponse, err error) {
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

	return res, nil
}

func QueryPairsGRPC(cli liquiditytypes.QueryClient, height int64) (poolsRes []liquiditytypes.Pair, err error) {
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
