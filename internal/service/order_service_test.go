package service

import (
	"math/big"
	"testing"
	"time"
)

func TestCreateOrder(t *testing.T) {
	// setup market
	marketService := NewMarketService()
	marketService.CreateMarket("BTC", "USD", 8, 6)
	marketTicker := GetMarketTicker("BTC", "USD")
	market := marketService.GetMarket(marketTicker)

	// setup order
	orderService := NewOrderService(marketService)
	// create orders
	for i := 0; i < 1; i++ {
		order := Order{
			ID:        orderService.GetNextOrderID(),
			OrderType: BuyOrder,
			Size:      big.NewInt(1e8),
			Price:     big.NewInt(112_000e6 + int64(i)),
			CreatedAt: time.Now(),
			Market:    market,
		}
		orderService.CreateOrder(order, marketTicker)
	}
	for i := 0; i < 1; i++ {
		order := Order{
			ID:        orderService.GetNextOrderID(),
			OrderType: SellOrder,
			Size:      big.NewInt(1e8),
			Price:     big.NewInt(112_000e6 + int64(i)),
			CreatedAt: time.Now(),
			Market:    market,
		}
		orderService.CreateOrder(order, marketTicker)
	}

	orderService.PrintOrders(marketTicker)
	marketService.PrintMarkets(marketTicker)
}
