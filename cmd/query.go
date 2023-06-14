package cmd

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"
	liquiditytypes "github.com/crescent-network/crescent/v4/x/liquidity/types"
	marketmakertypes "github.com/crescent-network/crescent/v4/x/marketmaker/types"
	minttypes "github.com/crescent-network/crescent/v4/x/mint/types"
	"google.golang.org/grpc/metadata"
)

func QueryOrdersGRPC(cli liquiditytypes.QueryClient, pairId uint64, height int64) (orderRes []liquiditytypes.Order, err error) {
	req := liquiditytypes.QueryOrdersRequest{
		PairId: pairId,
		Pagination: &query.PageRequest{
			Limit: 100000,
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
	if FlagManualParams {
		return &marketmakertypes.QueryParamsResponse{Params: ManualParams}, nil
	}

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

func QueryMarketMakersGRPC(cli marketmakertypes.QueryClient, height int64) (mms []marketmakertypes.MarketMaker, err error) {
	if FlagManualMarketMakers {
		return ManualMarketMakers, nil
	}

	req := marketmakertypes.QueryMarketMakersRequest{
		Pagination: &query.PageRequest{
			Limit: 10000,
		},
	}

	var header metadata.MD

	res, err := cli.MarketMakers(
		metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10)),
		&req,
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return res.Marketmakers, nil
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
			Limit: 10000,
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
			Limit: 10000,
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
	pm.Common = &params.Common
	// TODO: temporary generate mock pairs
	//params.IncentivePairs = append(params.IncentivePairs, marketmakertypes.IncentivePair{
	//	PairId:          8,
	//	UpdateTime:      utils.ParseTime("2022-09-01T00:00:00Z"),
	//	IncentiveWeight: sdk.MustNewDecFromStr("0.9"),
	//	MaxSpread:       sdk.MustNewDecFromStr("0.006"),
	//	MinWidth:        sdk.MustNewDecFromStr("0.001"),
	//	MinDepth:        sdk.NewInt(600000000000000000),
	//})

	// handle incentive pair's update time
	iMap := make(map[uint64]*marketmakertypes.IncentivePair)
	for _, pair := range params.IncentivePairs {
		if pair.UpdateTime.Before(*blockTime) {
			iMap[pair.PairId] = &pair
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

func GetMarketMakersMap(marketMakers []marketmakertypes.MarketMaker) (marketMakersMap map[uint64]map[string]*marketmakertypes.MarketMaker) {
	marketMakersMap = map[uint64]map[string]*marketmakertypes.MarketMaker{}
	for _, marketMaker := range marketMakers {
		if _, ok := marketMakersMap[marketMaker.PairId]; !ok {
			marketMakersMap[marketMaker.PairId] = map[string]*marketmakertypes.MarketMaker{}
		}
		marketMakersMap[marketMaker.PairId][marketMaker.Address] = &marketMaker
	}
	return
}
