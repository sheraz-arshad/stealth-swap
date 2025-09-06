package service

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var marketService *MarketService
var marketTicker string
var market Market
var orderService *OrderService

func setup() {
	marketService = NewMarketService()
	marketService.CreateMarket("BTC", "USD", 8, 6)
	marketTicker = GetMarketTicker("BTC", "USD")
	market = marketService.GetMarket(marketTicker)
	orderService = NewOrderService(marketService)
}

func TestCreateBuyOrder(t *testing.T) {
	setup()

	orderId := orderService.GetNextOrderID()
	size := big.NewInt(1e8)
	price := big.NewInt(111_000e6)
	order := Order{
		ID:         orderId,
		OrderType:  BuyOrder,
		Size:       size,
		Price:      price,
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	orderService.CreateOrder(order, marketTicker)
	orderExpected := orderService.GetActiveOrdersByMarketTicker(marketTicker)[0]
	assert.Equal(t, orderExpected.ID, orderId)
	assert.Equal(t, orderExpected.OrderType, BuyOrder)
	assert.Equal(t, orderExpected.Size, size)
	assert.Equal(t, orderExpected.Price, price)
	assert.Equal(t, orderExpected.SizeFilled, big.NewInt(0))
	assert.Equal(t, orderExpected.CreatedAt, order.CreatedAt)
	assert.Equal(t, orderExpected.Status, Open)
	assert.Equal(t, orderExpected.Market, market)

	assert.Equal(t, market.BuyLiquidityInBaseToken, size)
	assert.Equal(t, market.SellLiquidityInBaseToken, big.NewInt(0))
}

func TestCreateSellOrder(t *testing.T) {
	setup()

	orderId := orderService.GetNextOrderID()
	size := big.NewInt(1e8)
	price := big.NewInt(112_000e6)
	order := Order{
		ID:         orderId,
		OrderType:  SellOrder,
		Size:       size,
		Price:      price,
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	orderService.CreateOrder(order, marketTicker)
	orderExpected := orderService.GetActiveOrdersByMarketTicker(marketTicker)[0]
	assert.Equal(t, orderExpected.ID, orderId)
	assert.Equal(t, orderExpected.OrderType, SellOrder)
	assert.Equal(t, orderExpected.Size, size)
	assert.Equal(t, orderExpected.Price, price)
	assert.Equal(t, orderExpected.SizeFilled, big.NewInt(0))
	assert.Equal(t, orderExpected.CreatedAt, order.CreatedAt)
	assert.Equal(t, orderExpected.Status, Open)
	assert.Equal(t, orderExpected.Market, market)

	assert.Equal(t, market.BuyLiquidityInBaseToken, big.NewInt(0))
	assert.Equal(t, market.SellLiquidityInBaseToken, size)
}

func TestFillOrder(t *testing.T) {
	setup()
	// add buy liquidity
	for i := 0; i < 2; i++ {
		order := Order{
			ID:         orderService.GetNextOrderID(),
			OrderType:  BuyOrder,
			Size:       big.NewInt(1e8),
			Price:      big.NewInt(111_000e6 - int64(i*2000e6)),
			SizeFilled: big.NewInt(0),
			CreatedAt:  time.Now(),
			Status:     Open,
			Market:     market,
		}
		orderService.CreateOrder(order, marketTicker)
	}

	// fmt.Println(market.BuyLiquidityInBaseToken)
	// fmt.Println(market.SellLiquidityInBaseToken)

	// add sell liquidity
	for i := 0; i < 2; i++ {
		order := Order{
			ID:         orderService.GetNextOrderID(),
			OrderType:  SellOrder,
			Size:       big.NewInt(1e8),
			Price:      big.NewInt(112_000e6 + int64(i*2000e6)),
			SizeFilled: big.NewInt(0),
			CreatedAt:  time.Now(),
			Status:     Open,
			Market:     market,
		}
		orderService.CreateOrder(order, marketTicker)
	}

	// fmt.Println(market.BuyLiquidityInBaseToken)
	// fmt.Println(market.SellLiquidityInBaseToken)

	// fill order
	order := Order{
		ID:         orderService.GetNextOrderID(),
		OrderType:  SellOrder,
		Size:       big.NewInt(2e8),
		Price:      big.NewInt(112_000e6),
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	orderService.FillOrder(order, marketTicker)

	// all buy side is filled
	activeOrders := orderService.GetActiveOrdersByMarketTicker(marketTicker)
	for _, activeOrder := range activeOrders {
		assert.Equal(t, activeOrder.Status, Open)
		assert.Equal(t, activeOrder.SizeFilled, big.NewInt(0))
		assert.Equal(t, activeOrder.OrderType, SellOrder)
	}

	inActiveOrders := orderService.GetInActiveOrdersByMarketTicker(marketTicker)
	for _, inActiveOrder := range inActiveOrders {
		assert.Equal(t, inActiveOrder.Status, Filled)
		assert.Equal(t, inActiveOrder.SizeFilled, inActiveOrder.Size)
	}

	assert.Equal(t, market.BuyLiquidityInBaseToken.String(), big.NewInt(0).String())
	assert.Equal(t, market.SellLiquidityInBaseToken, big.NewInt(2e8))
}
