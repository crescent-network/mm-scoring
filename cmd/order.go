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
	utils "github.com/crescent-network/crescent/v5/types"
	"github.com/crescent-network/crescent/v5/x/liquidity/amm"
	liquiditytypes "github.com/crescent-network/crescent/v5/x/liquidity/types"
	marketmakertypes "github.com/crescent-network/crescent/v5/x/marketmaker/types"
)

type Result struct {
	Orders []liquiditytypes.Order

	MidPrice sdk.Dec
	Spread   sdk.Dec

	AskWidth sdk.Dec
	BidWidth sdk.Dec

	AskDepth sdk.Int
	BidDepth sdk.Int

	AskMaxPrice sdk.Dec
	AskMinPrice sdk.Dec
	BidMaxPrice sdk.Dec
	BidMinPrice sdk.Dec

	CBid sdk.Dec
	CAsk sdk.Dec
	CMin sdk.Dec // min(CBid, CAsk)

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
		AskDepth:    sdk.ZeroInt(),
		BidDepth:    sdk.ZeroInt(),
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
	// 2-sided liquidity required
	if len(r.Orders) < 2 {
		// TODO: test coverage
		r.Live = false
		r.CMin = sdk.ZeroDec()
		return r
	}

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
			r.BidDepth = r.BidDepth.Add(order.OpenAmount)
			if order.Price.GTE(r.BidMaxPrice) {
				r.BidMaxPrice = order.Price
			}
			if r.BidMinPrice.IsZero() || order.Price.LTE(r.BidMinPrice) {
				r.BidMinPrice = order.Price
			}
		} else if order.Direction == liquiditytypes.OrderDirectionSell { // ASK
			r.AskCount += 1
			r.AskDepth = r.AskDepth.Add(order.OpenAmount)
			if order.Price.GTE(r.AskMaxPrice) {
				r.AskMaxPrice = order.Price
			}
			if r.AskMinPrice.IsZero() || order.Price.LTE(r.AskMinPrice) {
				r.AskMinPrice = order.Price
			}
		}
	}
	// if bidMaxPrice or adkMinPrice is zero, we can't calculate mid price, set c_min to zero and return here
	if r.BidMaxPrice.IsZero() || r.AskMinPrice.IsZero() {
		r.Live = false
		r.CMin = sdk.ZeroDec()
		return r
	}
	// calc mid price, (BidMaxPrice + AskMinPrice)/2
	r.MidPrice = r.BidMaxPrice.Add(r.AskMinPrice).QuoTruncate(sdk.NewDec(2))
	r.Spread = r.AskMinPrice.Sub(r.BidMaxPrice).QuoTruncate(r.MidPrice)
	r.AskWidth = r.AskMaxPrice.Sub(r.AskMinPrice).QuoTruncate(r.MidPrice)
	r.BidWidth = r.BidMaxPrice.Sub(r.BidMinPrice).QuoTruncate(r.MidPrice)

	// Spread should be larger than 0, equal or smaller than MaxSpread
	if r.Spread.GT(pair.MaxSpread) || r.Spread.IsZero() {
		r.Live = false
		r.CMin = sdk.ZeroDec()
		return r
	}

	// Minimum allowable price difference of high and low on both side of orders
	if sdk.MinDec(r.AskWidth, r.BidWidth).LT(pair.MinWidth) {
		r.Live = false
		r.CMin = sdk.ZeroDec()
		return r
	}

	if sdk.MinInt(r.AskDepth, r.BidDepth).LT(pair.MinDepth) {
		// TODO: test coverage
		r.Live = false
		r.CMin = sdk.ZeroDec()
		return r
	}

	for _, order := range r.Orders {
		// skip orders which has not available status
		if order.Status != liquiditytypes.OrderStatusNotExecuted &&
			order.Status != liquiditytypes.OrderStatusNotMatched &&
			order.Status != liquiditytypes.OrderStatusPartiallyMatched {
			// TODO: test coverage
			continue
		}

		// TODO: need to check if the 2-sided order placed
		// 2-sided liquidity required, temporary panic, need to delete
		if order.Price.Equal(r.MidPrice) {
			// TODO: test coverage
			panic("WIP debugging, mid price should not be equal to order price")

		}
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
	// TODO: test coverage
	if r.CMin.IsZero() {
		r.Live = false
		return r
	}

	r.Live = true
	return r
}

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

func TimeToHour(timestamp *time.Time) (hour int) {
	hour = timestamp.Day() * 24
	hour += timestamp.Hour()
	return hour
}

func MMOrder(pair liquiditytypes.Pair, height int64, blockTime *time.Time, params *liquiditytypes.Params, msg *MsgMMOrder) (orders []liquiditytypes.Order, err error) {
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

	var buyTicks, sellTicks []MMOrderTick
	offerBaseCoin := sdk.NewInt64Coin(pair.BaseCoinDenom, 0)
	offerQuoteCoin := sdk.NewInt64Coin(pair.QuoteCoinDenom, 0)
	if msg.BuyAmount.IsPositive() {
		buyTicks = MMOrderTicks(
			liquiditytypes.OrderDirectionBuy, msg.MinBuyPrice, msg.MaxBuyPrice, msg.BuyAmount, maxNumTicks, tickPrec)
		for _, tick := range buyTicks {
			offerQuoteCoin = offerQuoteCoin.AddAmount(tick.OfferCoinAmount)
		}
	}
	if msg.SellAmount.IsPositive() {
		sellTicks = MMOrderTicks(
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

func GenerateMockOrders(pair liquiditytypes.Pair, incentivePair marketmakertypes.IncentivePair, height int64, blockTime *time.Time, params *liquiditytypes.Params, mmMap map[string]*MM) (orders []liquiditytypes.Order, err error) {
	r := rand.New(rand.NewSource(0))

	mms := []string{
		"cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk",
		"cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev",
		"cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k",
	}
	for _, mm := range mms {
		if _, ok := mmMap[mm]; !ok {
			mmMap[mm] = &MM{
				Address: mm,
				PairId:  pair.Id,
			}
		}
	}

	tickPrec := int(params.TickPrecision)

	for mm, _ := range mmMap {
		orderOrNot := rand.Intn(2)
		if orderOrNot == 1 {
			continue
		}

		sellAmount := incentivePair.MinDepth.MulRaw(int64(params.MaxNumMarketMakingOrderTicks)).ToDec().Mul(
			utils.RandomDec(r, utils.ParseDec("0.95"), utils.ParseDec("1.45"))).TruncateInt()

		buyAmount := incentivePair.MinDepth.MulRaw(int64(params.MaxNumMarketMakingOrderTicks)).ToDec().Mul(
			utils.RandomDec(r, utils.ParseDec("0.95"), utils.ParseDec("1.45"))).TruncateInt()

		simtypes.RandomDecAmount(r, sdk.NewDecWithPrec(1, 2))

		msg := MsgMMOrder{
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

// ============================ legacy crescent v4 mm order ============================

// MsgMMOrder defines an SDK message for making a MM(market making) order, v4 legacy
type MsgMMOrder struct {
	// orderer specifies the bech32-encoded address that makes an order
	Orderer string `protobuf:"bytes,1,opt,name=orderer,proto3" json:"orderer,omitempty"`
	// pair_id specifies the pair id
	PairId uint64 `protobuf:"varint,2,opt,name=pair_id,json=pairId,proto3" json:"pair_id,omitempty"`
	// max_sell_price specifies the maximum sell price
	MaxSellPrice sdk.Dec `protobuf:"bytes,3,opt,name=max_sell_price,json=maxSellPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"max_sell_price"`
	// min_sell_price specifies the minimum sell price
	MinSellPrice sdk.Dec `protobuf:"bytes,4,opt,name=min_sell_price,json=minSellPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"min_sell_price"`
	// sell_amount specifies the total amount of base coin of sell orders
	SellAmount sdk.Int `protobuf:"bytes,5,opt,name=sell_amount,json=sellAmount,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"sell_amount"`
	// max_buy_price specifies the maximum buy price
	MaxBuyPrice sdk.Dec `protobuf:"bytes,6,opt,name=max_buy_price,json=maxBuyPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"max_buy_price"`
	// min_buy_price specifies the minimum buy price
	MinBuyPrice sdk.Dec `protobuf:"bytes,7,opt,name=min_buy_price,json=minBuyPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"min_buy_price"`
	// buy_amount specifies the total amount of base coin of buy orders
	BuyAmount sdk.Int `protobuf:"bytes,8,opt,name=buy_amount,json=buyAmount,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"buy_amount"`
	// order_lifespan specifies the order lifespan
	OrderLifespan time.Duration `protobuf:"bytes,9,opt,name=order_lifespan,json=orderLifespan,proto3,stdduration" json:"order_lifespan"`
}

// NewMsgMMOrder creates a new MsgMMOrder.
func NewMsgMMOrder(
	orderer sdk.AccAddress,
	pairId uint64,
	maxSellPrice, minSellPrice sdk.Dec, sellAmt sdk.Int,
	maxBuyPrice, minBuyPrice sdk.Dec, buyAmt sdk.Int,
	orderLifespan time.Duration,
) *MsgMMOrder {
	return &MsgMMOrder{
		Orderer:       orderer.String(),
		PairId:        pairId,
		MaxSellPrice:  maxSellPrice,
		MinSellPrice:  minSellPrice,
		SellAmount:    sellAmt,
		MaxBuyPrice:   maxBuyPrice,
		MinBuyPrice:   minBuyPrice,
		BuyAmount:     buyAmt,
		OrderLifespan: orderLifespan,
	}
}

func (msg MsgMMOrder) Route() string { return liquiditytypes.RouterKey }

func (msg MsgMMOrder) Type() string { return liquiditytypes.TypeMsgMMOrder }

func (msg MsgMMOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Orderer); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid orderer address: %v", err)
	}
	if msg.PairId == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "pair id must not be 0")
	}
	if msg.SellAmount.IsZero() && msg.BuyAmount.IsZero() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "sell amount and buy amount must not be zero at the same time")
	}
	if !msg.SellAmount.IsZero() {
		if msg.SellAmount.LT(amm.MinCoinAmount) {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "sell amount %s is smaller than the min amount %s", msg.SellAmount, amm.MinCoinAmount)
		}
		if !msg.MaxSellPrice.IsPositive() {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "max sell price must be positive: %s", msg.MaxSellPrice)
		}
		if !msg.MinSellPrice.IsPositive() {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "min sell price must be positive: %s", msg.MinSellPrice)
		}
		if msg.MinSellPrice.GT(msg.MaxSellPrice) {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "max sell price must not be lower than min sell price")
		}
	}
	if !msg.BuyAmount.IsZero() {
		if msg.BuyAmount.LT(amm.MinCoinAmount) {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "buy amount %s is smaller than the min amount %s", msg.BuyAmount, amm.MinCoinAmount)
		}
		if !msg.MinBuyPrice.IsPositive() {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "min buy price must be positive: %s", msg.MinBuyPrice)
		}
		if !msg.MaxBuyPrice.IsPositive() {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "max buy price must be positive: %s", msg.MaxBuyPrice)
		}
		if msg.MinBuyPrice.GT(msg.MaxBuyPrice) {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "max buy price must not be lower than min buy price")
		}
	}
	if msg.OrderLifespan < 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "order lifespan must not be negative: %s", msg.OrderLifespan)
	}
	return nil
}

//func (msg MsgMMOrder) GetSignBytes() []byte {
//	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
//}

func (msg MsgMMOrder) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Orderer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgMMOrder) GetOrderer() sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Orderer)
	if err != nil {
		panic(err)
	}
	return addr
}

// MMOrderTick holds information about each tick's price and amount of an MMOrder.
type MMOrderTick struct {
	OfferCoinAmount sdk.Int
	Price           sdk.Dec
	Amount          sdk.Int
}

// MMOrderTicks returns fairly distributed tick information with given parameters.
func MMOrderTicks(dir liquiditytypes.OrderDirection, minPrice, maxPrice sdk.Dec, amt sdk.Int, maxNumTicks, tickPrec int) (ticks []MMOrderTick) {
	ammDir := amm.OrderDirection(dir)
	if minPrice.Equal(maxPrice) {
		return []MMOrderTick{{OfferCoinAmount: amm.OfferCoinAmount(ammDir, minPrice, amt), Price: minPrice, Amount: amt}}
	}
	gap := maxPrice.Sub(minPrice).QuoInt64(int64(maxNumTicks - 1))
	switch dir {
	case liquiditytypes.OrderDirectionBuy:
		var prevP sdk.Dec
		for i := 0; i < maxNumTicks-1; i++ {
			p := amm.PriceToDownTick(minPrice.Add(gap.MulInt64(int64(i))), tickPrec)
			if prevP.IsNil() || !p.Equal(prevP) {
				ticks = append(ticks, MMOrderTick{
					Price: p,
				})
				prevP = p
			}
		}
		tickAmt := amt.QuoRaw(int64(len(ticks) + 1))
		for i := range ticks {
			ticks[i].Amount = tickAmt
			ticks[i].OfferCoinAmount = amm.OfferCoinAmount(ammDir, ticks[i].Price, ticks[i].Amount)
			amt = amt.Sub(tickAmt)
		}
		ticks = append(ticks, MMOrderTick{
			OfferCoinAmount: amm.OfferCoinAmount(ammDir, maxPrice, amt),
			Price:           maxPrice,
			Amount:          amt,
		})
	case liquiditytypes.OrderDirectionSell:
		var prevP sdk.Dec
		for i := 0; i < maxNumTicks-1; i++ {
			p := amm.PriceToUpTick(maxPrice.Sub(gap.MulInt64(int64(i))), tickPrec)
			if prevP.IsNil() || !p.Equal(prevP) {
				ticks = append(ticks, MMOrderTick{
					Price: p,
				})
				prevP = p
			}
		}
		tickAmt := amt.QuoRaw(int64(len(ticks) + 1))
		for i := range ticks {
			ticks[i].Amount = tickAmt
			ticks[i].OfferCoinAmount = amm.OfferCoinAmount(ammDir, ticks[i].Price, ticks[i].Amount)
			amt = amt.Sub(tickAmt)
		}
		ticks = append(ticks, MMOrderTick{
			OfferCoinAmount: amm.OfferCoinAmount(ammDir, minPrice, amt),
			Price:           minPrice,
			Amount:          amt,
		})
	}
	return
}
