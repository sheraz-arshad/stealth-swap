package service

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type OrderBook struct {
	Orders         []Order
	LastPrice      *big.Int
	SellIndex      int
	BuyIndex       int
	InActiveOrders []Order
}

type OrderService struct {
	OrderBooks    map[string]OrderBook
	marketService *MarketService
	orderID       int64
}

type OrderType string

const (
	BuyOrder  OrderType = "BUY"
	SellOrder OrderType = "SELL"
)

type OrderStatus string

const (
	Open   OrderStatus = "OPEN"
	Closed OrderStatus = "CLOSED"
	Filled OrderStatus = "FILLED"
)

type Order struct {
	ID   int64
	User common.Address
	OrderType
	Size       *big.Int
	Price      *big.Int
	SizeFilled *big.Int
	CreatedAt  time.Time
	Status     OrderStatus
	Market     Market
}

func NewOrderService(marketService *MarketService) *OrderService {
	return &OrderService{
		OrderBooks:    make(map[string]OrderBook),
		marketService: marketService,
		orderID:       0,
	}
}

func (service *OrderService) CreateOrder(order Order, marketTicker string) {
	// insert new order in the list based on the price where the order before it should have higher price
	orderBook := service.OrderBooks[marketTicker]
	insertIdx := len(orderBook.Orders)

	for i, existingOrder := range orderBook.Orders {
		if order.Price.Cmp(existingOrder.Price) < 0 {
			insertIdx = i
			break
		}
	}
	orderBook.Orders = append(orderBook.Orders, Order{}) // extend slice
	copy(orderBook.Orders[insertIdx+1:], orderBook.Orders[insertIdx:])
	orderBook.Orders[insertIdx] = order

	if len(orderBook.Orders) == 1 {
		orderBook.LastPrice = order.Price
	} else {
		if order.OrderType == BuyOrder {
			orderBook.BuyIndex = orderBook.BuyIndex + 1
			if orderBook.SellIndex > 0 {
				orderBook.SellIndex = orderBook.SellIndex + 1
			}
		} else if order.OrderType == SellOrder && (insertIdx < orderBook.SellIndex || orderBook.SellIndex <= 0) {
			orderBook.SellIndex = insertIdx
		}
	}
	service.OrderBooks[marketTicker] = orderBook

	if order.OrderType == BuyOrder {
		service.marketService.UpdateLiquidity(
			marketTicker,
			order.Size,
			big.NewInt(0),
		)
	} else {
		service.marketService.UpdateLiquidity(
			marketTicker,
			big.NewInt(0),
			order.Size,
		)
	}
}

func (service *OrderService) FillOrder(order Order, marketTicker string) {
	orderBook := service.OrderBooks[marketTicker]
	amountRemaining := new(big.Int).Set(order.Size)

	for amountRemaining.Cmp(big.NewInt(0)) > 0 {
		if len(orderBook.Orders) == 0 {
			break
		}
		var makerIndex int
		if order.OrderType == BuyOrder {
			makerIndex = orderBook.SellIndex
		} else {
			makerIndex = orderBook.BuyIndex
		}

		if makerIndex < 0 || makerIndex >= len(orderBook.Orders) {
			break
		}

		makerOrder := orderBook.Orders[makerIndex]
		amountAvailable := new(big.Int).Sub(makerOrder.Size, makerOrder.SizeFilled)
		if amountAvailable.Cmp(amountRemaining) >= 0 {
			order.SizeFilled.Add(order.SizeFilled, amountRemaining)
			makerOrder.SizeFilled.Add(makerOrder.SizeFilled, amountRemaining)
			amountRemaining = big.NewInt(0)
		} else {
			order.SizeFilled.Add(order.SizeFilled, amountAvailable)
			amountRemaining.Sub(amountRemaining, amountAvailable)
			makerOrder.SizeFilled.Add(makerOrder.SizeFilled, amountAvailable)
		}

		if makerOrder.SizeFilled.Cmp(makerOrder.Size) == 0 {
			makerOrder.Status = Filled
			orderBook.InActiveOrders = append(orderBook.InActiveOrders, makerOrder)

			orderBook.Orders = append(
				orderBook.Orders[:makerIndex],
				orderBook.Orders[makerIndex+1:]...)

			if order.OrderType == SellOrder {
				orderBook.SellIndex = orderBook.SellIndex - 1
				orderBook.BuyIndex = orderBook.BuyIndex - 1
			}
		} else {
			orderBook.Orders[makerIndex] = makerOrder
		}
		orderBook.LastPrice = makerOrder.Price
	}
	order.Status = Filled
	orderBook.InActiveOrders = append(orderBook.InActiveOrders, order)
	service.OrderBooks[marketTicker] = orderBook

	if order.OrderType == BuyOrder {
		service.marketService.UpdateLiquidity(
			marketTicker,
			big.NewInt(0),
			new(big.Int).Neg(order.SizeFilled),
		)
	} else {
		service.marketService.UpdateLiquidity(
			marketTicker,
			new(big.Int).Neg(order.SizeFilled),
			big.NewInt(0),
		)
	}
}

func (service *OrderService) GetActiveOrdersByMarketTicker(marketTicker string) []Order {
	return append([]Order{}, service.OrderBooks[marketTicker].Orders...)
}

func (service *OrderService) GetInActiveOrdersByMarketTicker(marketTicker string) []Order {
	return append([]Order{}, service.OrderBooks[marketTicker].InActiveOrders...)
}

func (service *OrderService) GetNextOrderID() int64 {
	service.orderID++
	return service.orderID
}

// PrintOrders prints all orders to console in a formatted way
func (service *OrderService) PrintActiveOrders(marketTicker string) {
	fmt.Println("=== ACTIVE ORDERS ===")

	market := service.marketService.GetMarket(marketTicker)
	quoteMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.QuoteTokenDecimals)), nil)
	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.BaseTokenDecimals)), nil)

	orderBook := service.OrderBooks[marketTicker]
	fmt.Println("Last Price:", new(big.Int).Div(orderBook.LastPrice, quoteMultiplier))
	fmt.Println("First Buy Index:", orderBook.BuyIndex)
	fmt.Println("First Sell Index:", orderBook.SellIndex)

	if len(orderBook.Orders) == 0 {
		fmt.Println("No orders found")
		return
	}

	orders := append([]Order{}, orderBook.Orders...)
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Price.Cmp(orders[j].Price) > 0
	})

	for _, order := range orders {
		fmt.Printf(
			"Order ID:%d | Type: %s | Size: %d | Price: %d | Size Filled: %d | Market: %s | Time: %s | Status: %s\n",
			order.ID,
			order.OrderType,
			new(big.Int).Div(order.Size, baseMultiplier),
			new(big.Int).Div(order.Price, quoteMultiplier),
			new(big.Int).Div(order.SizeFilled, baseMultiplier),
			market.MarketTicker,
			order.CreatedAt.Format("15:04:05"),
			order.Status,
		)
	}
	fmt.Println("=============")
}

func (service *OrderService) PrintInActiveOrders(marketTicker string) {
	fmt.Println("=== INACTIVE ORDERS ===")

	market := service.marketService.GetMarket(marketTicker)
	quoteMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.QuoteTokenDecimals)), nil)
	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.BaseTokenDecimals)), nil)

	orderBook := service.OrderBooks[marketTicker]
	if len(orderBook.InActiveOrders) == 0 {
		fmt.Println("No inactive orders found")
		return
	}

	orders := append([]Order{}, orderBook.InActiveOrders...)
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})

	for _, order := range orders {
		fmt.Printf(
			"Order ID:%d | Type: %s | Size: %d | Price: %d | Size Filled: %d | Market: %s | Time: %s | Status: %s\n",
			order.ID,
			order.OrderType,
			new(big.Int).Div(order.Size, baseMultiplier),
			new(big.Int).Div(order.Price, quoteMultiplier),
			new(big.Int).Div(order.SizeFilled, baseMultiplier),
			market.MarketTicker,
			order.CreatedAt.Format("15:04:05"),
			order.Status,
		)
	}
	fmt.Println("=============")
}
