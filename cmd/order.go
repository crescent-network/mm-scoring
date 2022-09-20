package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/crescent-network/crescent/v3/x/liquidity/amm"
	liquiditytypes "github.com/crescent-network/crescent/v3/x/liquidity/types"
)

// TODO: order checking status, expired as function on this?
// TODO: Calc Spread from order list of the height of a pair
// TODO: Distance, midPrice, both side, min
// testcode with input orderlist
// GET C_mt, summation of C_mt for share
// Spread, Distance, Width
// SET params as json file

type Result struct {
	Orders []liquiditytypes.Order

	MidPrice sdk.Dec
	Spread   sdk.Dec

	AskWidth    sdk.Dec
	BidWidth    sdk.Dec
	AskQuantity sdk.Int
	BidQuantity sdk.Int

	AskMaxPrice sdk.Dec
	AskMinPrice sdk.Dec
	BidMaxPrice sdk.Dec
	BidMinPrice sdk.Dec

	CBid sdk.Dec // TODO: To be deleted
	CAsk sdk.Dec // TODO: To be deleted
	CMin sdk.Dec // min(CBid, CAsk)

	// TODO: MinWidth, MinDepth, MaxSpread

	// TODO: live Uptime
	Live bool

	BidCount           int
	AskCount           int
	RemCount           int
	InvalidStatusCount int
	TotalCount         int
}

func NewResult() (result *Result) {
	return &Result{
		Orders: []liquiditytypes.Order{},

		MidPrice:    sdk.ZeroDec(),
		AskWidth:    sdk.ZeroDec(),
		BidWidth:    sdk.ZeroDec(),
		AskQuantity: sdk.ZeroInt(),
		BidQuantity: sdk.ZeroInt(),
		AskMaxPrice: sdk.ZeroDec(),
		AskMinPrice: sdk.ZeroDec(),
		BidMaxPrice: sdk.ZeroDec(),
		BidMinPrice: sdk.ZeroDec(),

		CBid: sdk.ZeroDec(),
		CAsk: sdk.ZeroDec(),
		CMin: sdk.ZeroDec(),

		// info field
		Spread:             sdk.ZeroDec(),
		BidCount:           0,
		AskCount:           0,
		RemCount:           0,
		InvalidStatusCount: 0,
		TotalCount:         0,
	}
}

func (r Result) String() (str string) {
	s := reflect.ValueOf(&r).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		lineStr := fmt.Sprintf("%s %s = %v\n",
			typeOfT.Field(i).Name, f.Type(), f.Interface())
		str = str + lineStr
	}
	return
}

func SetResult(r *Result, pm ParamsMap, pairId uint64) *Result {
	pair := pm.IncentivePairsMap[pairId]
	for _, order := range r.Orders {
		r.TotalCount += 1

		// skip orders which has not available status
		if order.Status != liquiditytypes.OrderStatusNotExecuted &&
			order.Status != liquiditytypes.OrderStatusNotMatched &&
			order.Status != liquiditytypes.OrderStatusPartiallyMatched {
			r.InvalidStatusCount += 1
			continue
		}
		// skip orders which is not over MinOpenRatio, and over MinOpenRatio of MinDepth from param
		if order.OpenAmount.ToDec().QuoTruncate(order.Amount.ToDec()).LTE(pm.Common.MinOpenRatio) && order.OpenAmount.LT(
			pair.MinDepth.ToDec().MulTruncate(pm.Common.MinOpenDepthRatio).TruncateInt()) {
			r.RemCount += 1
			continue
		}
		if order.Direction == liquiditytypes.OrderDirectionBuy { // BID
			r.BidCount += 1
			r.BidQuantity = r.BidQuantity.Add(order.OpenAmount)
			if order.Price.GTE(r.BidMaxPrice) {
				r.BidMaxPrice = order.Price
			}
			if r.BidMinPrice.IsZero() || order.Price.LTE(r.BidMinPrice) {
				r.BidMinPrice = order.Price
			}
		} else if order.Direction == liquiditytypes.OrderDirectionSell { // ASK
			r.AskCount += 1
			r.AskQuantity = r.AskQuantity.Add(order.OpenAmount)
			if order.Price.GTE(r.AskMaxPrice) {
				r.AskMaxPrice = order.Price
			}
			if r.AskMinPrice.IsZero() || order.Price.LTE(r.AskMinPrice) {
				r.AskMinPrice = order.Price
			}
		}
	}
	if r.BidMaxPrice.IsZero() || r.AskMinPrice.IsZero() {
		return nil
	}
	// calc mid price, (BidMaxPrice + AskMinPrice)/2
	r.MidPrice = r.BidMaxPrice.Add(r.AskMinPrice).QuoTruncate(sdk.NewDec(2))
	r.Spread = r.AskMinPrice.Sub(r.BidMaxPrice).QuoTruncate(r.MidPrice)
	r.AskWidth = r.AskMaxPrice.Sub(r.AskMinPrice).QuoTruncate(r.MidPrice)
	r.BidWidth = r.BidMaxPrice.Sub(r.BidMinPrice).QuoTruncate(r.MidPrice)

	for _, order := range r.Orders {
		if order.Direction == liquiditytypes.OrderDirectionSell {
			askD := order.Price.Sub(r.MidPrice).QuoTruncate(r.MidPrice)
			r.CAsk = r.CAsk.Add(order.OpenAmount.ToDec().QuoTruncate(askD.Power(2)))
		} else if order.Direction == liquiditytypes.OrderDirectionBuy {
			bidD := r.MidPrice.Sub(order.Price).QuoTruncate(r.MidPrice)
			r.CBid = r.CBid.Add(order.OpenAmount.ToDec().QuoTruncate(bidD.Power(2)))
		}
	}
	r.CMin = sdk.MinDec(r.CAsk, r.CBid)

	// invalid orders
	if r.CMin == sdk.ZeroDec() {
		r.Live = false
		return r
	}

	// Score is calculated for orders with spread smaller than MaxSpread
	if r.Spread.GT(pair.MaxSpread) {
		r.Live = false
		return r
	}

	// TODO: need to check
	// Minimum allowable price difference of high and low on both side of orders
	if sdk.MinDec(r.AskWidth, r.BidWidth).LT(pair.MinWidth) {
		r.Live = false
		fmt.Println(r.AskWidth, r.BidWidth, pair.MinWidth)
		fmt.Println("MIN WIDTH")
		return r
	}

	r.Live = true
	return r
	// TODO: checking order tick cap validity
}

// TODO: live calculation from map OrderMapByHeight

func output(data interface{}, filename string) {
	var p []byte
	p, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile(filename, p, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// TODO: add test case
func TimeToHour(timestamp *time.Time) (hour int) {
	hour = timestamp.Day() * 24
	hour += timestamp.Hour()
	return hour
}

func GenWithdrawFeeRate(r *rand.Rand) sdk.Dec {
	// TODO: get mid price, get Max, Min
	return simtypes.RandomDecAmount(r, sdk.NewDecWithPrec(1, 2))
}

func MMOrder(pair liquiditytypes.Pair, height int64, blockTime *time.Time, params *liquiditytypes.Params, msg *liquiditytypes.MsgMMOrder) (orders []liquiditytypes.Order, err error) {
	tickPrec := int(params.TickPrecision)

	if msg.SellAmount.IsPositive() {
		if !amm.PriceToDownTick(msg.MinSellPrice, tickPrec).Equal(msg.MinSellPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceNotOnTicks, "min sell price is not on ticks")
		}
		if !amm.PriceToDownTick(msg.MaxSellPrice, tickPrec).Equal(msg.MaxSellPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceNotOnTicks, "max sell price is not on ticks")
		}
	}
	if msg.BuyAmount.IsPositive() {
		if !amm.PriceToDownTick(msg.MinBuyPrice, tickPrec).Equal(msg.MinBuyPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceNotOnTicks, "min buy price is not on ticks")
		}
		if !amm.PriceToDownTick(msg.MaxBuyPrice, tickPrec).Equal(msg.MaxBuyPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceNotOnTicks, "max buy price is not on ticks")
		}
	}

	var lowestPrice, highestPrice sdk.Dec
	if pair.LastPrice != nil {
		lowestPrice, highestPrice = liquiditytypes.PriceLimits(*pair.LastPrice, params.MaxPriceLimitRatio, tickPrec)
	} else {
		lowestPrice = amm.LowestTick(tickPrec)
		highestPrice = amm.HighestTick(tickPrec)
	}

	if msg.SellAmount.IsPositive() {
		if msg.MinSellPrice.LT(lowestPrice) || msg.MinSellPrice.GT(highestPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceOutOfRange, "min sell price is out of range [%s, %s]", lowestPrice, highestPrice)
		}
		if msg.MaxSellPrice.LT(lowestPrice) || msg.MaxSellPrice.GT(highestPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceOutOfRange, "max sell price is out of range [%s, %s]", lowestPrice, highestPrice)
		}
	}
	if msg.BuyAmount.IsPositive() {
		if msg.MinBuyPrice.LT(lowestPrice) || msg.MinBuyPrice.GT(highestPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceOutOfRange, "min buy price is out of range [%s, %s]", lowestPrice, highestPrice)
		}
		if msg.MaxBuyPrice.LT(lowestPrice) || msg.MaxBuyPrice.GT(highestPrice) {
			return nil, sdkerrors.Wrapf(liquiditytypes.ErrPriceOutOfRange, "max buy price is out of range [%s, %s]", lowestPrice, highestPrice)
		}
	}

	maxNumTicks := int(params.MaxNumMarketMakingOrderTicks)

	var buyTicks, sellTicks []liquiditytypes.MMOrderTick
	offerBaseCoin := sdk.NewInt64Coin(pair.BaseCoinDenom, 0)
	offerQuoteCoin := sdk.NewInt64Coin(pair.QuoteCoinDenom, 0)
	if msg.BuyAmount.IsPositive() {
		buyTicks = liquiditytypes.MMOrderTicks(
			liquiditytypes.OrderDirectionBuy, msg.MinBuyPrice, msg.MaxBuyPrice, msg.BuyAmount, maxNumTicks, tickPrec)
		for _, tick := range buyTicks {
			offerQuoteCoin = offerQuoteCoin.AddAmount(tick.OfferCoinAmount)
		}
	}
	if msg.SellAmount.IsPositive() {
		sellTicks = liquiditytypes.MMOrderTicks(
			liquiditytypes.OrderDirectionSell, msg.MinSellPrice, msg.MaxSellPrice, msg.SellAmount, maxNumTicks, tickPrec)
		for _, tick := range sellTicks {
			offerBaseCoin = offerBaseCoin.AddAmount(tick.OfferCoinAmount)
		}
	}

	orderer := msg.GetOrderer()

	maxOrderLifespan := params.MaxOrderLifespan
	if msg.OrderLifespan > maxOrderLifespan {
		return nil, sdkerrors.Wrapf(
			liquiditytypes.ErrTooLongOrderLifespan, "%s is longer than %s", msg.OrderLifespan, maxOrderLifespan)
	}

	expireAt := blockTime.Add(msg.OrderLifespan)
	lastOrderId := pair.LastOrderId

	var orderIds []uint64
	for _, tick := range buyTicks {
		lastOrderId++
		offerCoin := sdk.NewCoin(pair.QuoteCoinDenom, tick.OfferCoinAmount)
		order := liquiditytypes.NewOrder(
			liquiditytypes.OrderTypeMM, lastOrderId, pair, orderer,
			offerCoin, tick.Price, tick.Amount, expireAt, height)
		orders = append(orders, order)
		orderIds = append(orderIds, order.Id)
	}
	for _, tick := range sellTicks {
		lastOrderId++
		offerCoin := sdk.NewCoin(pair.BaseCoinDenom, tick.OfferCoinAmount)
		order := liquiditytypes.NewOrder(
			liquiditytypes.OrderTypeMM, lastOrderId, pair, orderer,
			offerCoin, tick.Price, tick.Amount, expireAt, height)
		orders = append(orders, order)
		orderIds = append(orderIds, order.Id)
	}
	return
}
