package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"

	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	chain "github.com/crescent-network/crescent/v3/app"
	liquidityparams "github.com/crescent-network/crescent/v3/app/params"
	crecmd "github.com/crescent-network/crescent/v3/cmd/crescentd/cmd"
	liquiditytypes "github.com/crescent-network/crescent/v3/x/liquidity/types"
)

type Context struct {
	StartHeight int64
	LastHeight  int64
	SyncStatus  bool // TODO: delete?

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
	//RpcEndpoint        string
	//Dir                string
	// TODO: orderbook argument
}

var config = Config{
	StartHeight:  0,
	GrpcEndpoint: "127.0.0.1:9090",
	//RpcEndpoint:  "tcp://127.0.0.1:26657",
	//Dir:          "", // TODO: home
	// TODO: GRPC, RPC endpoint
}

func NewScoringCmd() *cobra.Command {
	crecmd.GetConfig()
	cmd := &cobra.Command{
		// TODO: grpc endpoint
		Use:  "mm-scoring [start-height]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			startHeight, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("parse start-Height: %w", err)
			}
			return Main(startHeight)
		},
	}
	return cmd
}

func Main(startHeight int64) error {

	var ctx Context
	ctx.Config = config
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

	fmt.Println("version :", "v0.1.0")
	fmt.Println("Input Start Height :", startHeight)

	//// checking start height
	//block := blockStore.LoadBlock(startHeight)
	//if block == nil {
	//	fmt.Println(startHeight, "is not available on this data")
	//	for i := 0; i < 1000000000000; i++ {
	//		block := blockStore.LoadBlock(int64(i))
	//		if block != nil {
	//			fmt.Println("available starting Height : ", i)
	//			break
	//		}
	//	}
	//	return nil
	//}
	//
	//// checking end height
	//if endHeight > blockStore.Height() {
	//	fmt.Println(endHeight, "is not available, Latest Height : ", blockStore.Height())
	//	return nil
	//}

	// Init app
	//encCfg := app.MakeEncodingConfig()
	//app := app.NewApp(log.NewNopLogger(), db, nil, false, map[int64]bool{}, "localnet", 0, encCfg, app.EmptyAppOptions{})
	//// should be load last height from v2 (sdk 0.45.*)
	//if err := app.LoadHeight(endHeight); err != nil {
	//	panic(err)
	//}

	//// Set tx index
	//store, err := tmdb.NewGoLevelDBWithOpts("data/tx_index", dir, &opt.Options{
	//	ErrorIfMissing: true,
	//	ReadOnly:       true,
	//})
	//if err != nil {
	//	return fmt.Errorf("open db: %w", err)
	//}
	//defer store.Close()
	// ============================ tx parsing logics ========================
	//txi := txkv.NewTxIndex(store)
	//txDecoder := encCfg.TxConfig.TxDecoder()
	// ============================ tx parsing logics ========================

	//ctx := app.BaseApp.NewContext(true, tmproto.Header{})
	//pairs := app.LiquidityKeeper.GetAllPairs(ctx)
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
	for i := startHeight; ; {
		_, err := QueryParamsGRPC(ctx.LiquidityClient, i)
		fmt.Println(i, err)
		if err != nil {
			time.Sleep(1000000000)
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
		i++
		// ============================ tx parsing logics ========================
		//// get block result
		//block := blockStore.LoadBlock(i)
		//
		//// iterate and parse txs of this block, ordered by txResult.Index 0 -> n
		//for _, tx := range block.Txs {
		//	txResult, err := txi.Get(tx.Hash())
		//	if err != nil {
		//		return fmt.Errorf("get tx index: %w", err)
		//	}
		//
		//	// pass if not succeeded tx
		//	if txResult.Result.Code != 0 {
		//		continue
		//	}
		//
		//	sdkTx, err := txDecoder(txResult.Tx)
		//	if err != nil {
		//		return fmt.Errorf("decode tx: %w", err)
		//	}
		//
		//	// indexing only targeted msg types
		//	for _, msg := range sdkTx.GetMsgs() {
		//		switch msg := msg.(type) {
		//		// TODO: filter only MM order type MMOrder, MMOrderCancel
		//		case *liquiditytypes.MsgLimitOrder:
		//			orderTxCount++
		//			mmOrderMap[i][msg.PairId] = append(mmOrderMap[i][msg.PairId], *msg)
		//			//fmt.Println(i, msg.Orderer, msg.Price, txResult.Result.Code, hex.EncodeToString(tx.Hash()))
		//		}
		//	}
		//}
		// ============================ tx parsing logics ========================
	}
	//jsonString, err := json.Marshal(OrderDataList)
	//fmt.Println(string(jsonString))
	//fmt.Println("finish", orderCount)
	//// TODO: analysis logic

	//for _, addrMap := range OrderMapByAddr {
	//	for _, heightMap := range addrMap {
	//		for _, orders := range heightMap {
	//			if len(orders) > 1 {
	//				fmt.Println("=====================")
	//				fmt.Printf("%#v\n", orders)
	//			}
	//		}
	//	}
	//}
	return nil
}

// TODO: query params for checking pruning

//func QueryPairs(app app.App, height int64) (poolsRes []liquiditytypes.Pair, err error) {
//	pairsReq := liquiditytypes.QueryPairsRequest{
//		Pagination: &query.PageRequest{
//			Limit: 100,
//		},
//	}
//
//	pairsRawData, err := pairsReq.Marshal()
//	if err != nil {
//		return nil, err
//	}
//
//	pairsRes := app.Query(abci.RequestQuery{
//		Path:   "/crescent.liquidity.v1beta1.Query/Pairs",
//		Data:   pairsRawData,
//		Height: height,
//		Prove:  false,
//	})
//	if pairsRes.Height != height {
//		fmt.Println(fmt.Errorf("pairs height error %d, %d", pairsRes.Height, height))
//	}
//	var pairsLive liquiditytypes.QueryPairsResponse
//	pairsLive.Unmarshal(pairsRes.Value)
//	return pairsLive.Pairs, nil
//}
//
//func QueryPools(app app.App, pairId uint64, height int64) (poolsRes []liquiditytypes.PoolResponse, err error) {
//	// Query Pool
//	poolsReq := liquiditytypes.QueryPoolsRequest{
//		PairId:   pairId,
//		Disabled: "false",
//		Pagination: &query.PageRequest{
//			Limit: 50,
//		},
//	}
//	dataPools, err := poolsReq.Marshal()
//	if err != nil {
//		return nil, err
//	}
//
//	resPool := app.Query(abci.RequestQuery{
//		Path:   "/crescent.liquidity.v1beta1.Query/Pools",
//		Data:   dataPools,
//		Height: height,
//		Prove:  false,
//	})
//	if resPool.Height != height {
//		fmt.Println(fmt.Errorf("pools height error %d, %d", resPool.Height, height))
//	}
//	var pools liquiditytypes.QueryPoolsResponse
//	pools.Unmarshal(resPool.Value)
//	return pools.Pools, nil
//}
//
//func QueryOrders(app app.App, pairId uint64, height int64) (poolsRes []liquiditytypes.Order, err error) {
//	a := liquiditytypes.QueryOrdersRequest{
//		PairId: pairId,
//		Pagination: &query.PageRequest{
//			Limit: 1000000,
//		},
//	}
//	data, err := a.Marshal()
//	if err != nil {
//		return nil, err
//	}
//
//	// Query Orders
//	res := app.Query(abci.RequestQuery{
//		Path:   "/crescent.liquidity.v1beta1.Query/Orders",
//		Data:   data,
//		Height: height,
//		Prove:  false,
//	})
//	if res.Height != height {
//		fmt.Println(fmt.Errorf("orders height error %d, %d", res.Height, height))
//	}
//	var orders liquiditytypes.QueryOrdersResponse
//	orders.Unmarshal(res.Value)
//	return orders.Orders, nil
//}

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
