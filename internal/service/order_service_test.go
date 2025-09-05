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
	// for i := 0; i < 2; i++ {
	// 	order := Order{
	// 		ID:         orderService.GetNextOrderID(),
	// 		OrderType:  SellOrder,
	// 		Size:       big.NewInt(1e8),
	// 		Price:      big.NewInt(112_000e6 + int64(i*2000e6)),
	// 		SizeFilled: big.NewInt(0),
	// 		CreatedAt:  time.Now(),
	// 		Status:     Open,
	// 		Market:     market,
	// 	}
	// 	orderService.CreateOrder(order, marketTicker)
	// }

	orderService.PrintActiveOrders(marketTicker)
	// orderService.PrintInActiveOrders(marketTicker)

	order := Order{
		ID:         orderService.GetNextOrderID(),
		OrderType:  SellOrder,
		Size:       big.NewInt(3e8),
		Price:      big.NewInt(112_000e6),
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	_ = order

	orderService.FillOrder(order, marketTicker)
	orderService.PrintActiveOrders(marketTicker)

	for i := 0; i < 2; i++ {
		order := Order{
			ID:         orderService.GetNextOrderID(),
			OrderType:  BuyOrder,
			Size:       big.NewInt(1e8),
			Price:      big.NewInt(100_000e6 - int64(i*2000e6)),
			SizeFilled: big.NewInt(0),
			CreatedAt:  time.Now(),
			Status:     Open,
			Market:     market,
		}
		orderService.CreateOrder(order, marketTicker)
	}
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

	orderService.PrintActiveOrders(marketTicker)
	// orderService.PrintInActiveOrders(marketTicker)
	// marketService.PrintMarkets(marketTicker)
}
