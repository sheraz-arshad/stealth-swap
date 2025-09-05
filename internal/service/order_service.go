package service

import (
	"fmt"
	"math/big"
	"time"
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
	ID int64
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
	insertIdx := 0
	orderBook := service.OrderBooks[marketTicker]

	for i, existingOrder := range orderBook.Orders {
		if existingOrder.Status == Closed || existingOrder.Status == Filled {
			continue
		}
		if order.Price.Cmp(existingOrder.Price) > 0 {
			insertIdx = i + 1
			break
		}
	}
	orderBook.Orders = append(orderBook.Orders, Order{}) // extend slice
	copy(orderBook.Orders[insertIdx+1:], orderBook.Orders[insertIdx:])
	orderBook.Orders[insertIdx] = order
	service.OrderBooks[marketTicker] = orderBook

	if len(orderBook.Orders) == 1 {
		orderBook.LastPrice = order.Price
		service.OrderBooks[marketTicker] = orderBook
	}

	if order.OrderType == BuyOrder && insertIdx > orderBook.BuyIndex {
		orderBook.BuyIndex = insertIdx
	} else if order.OrderType == SellOrder && insertIdx < orderBook.SellIndex {
		orderBook.SellIndex = insertIdx
	}

	if order.OrderType == BuyOrder {
		// liquidityAmount := new(big.Int)
		// // Step 1: Multiply amount * price
		// liquidityAmount.Mul(order.Amount, order.Price)

		// // Step 2: Multiply by 10^quoteTokenDecimals
		// quoteMultiplier := new(
		// 	big.Int,
		// ).Exp(big.NewInt(10), big.NewInt(int64(quoteTokenDecimals)), nil)
		// liquidityAmount.Mul(liquidityAmount, quoteMultiplier)

		// // Step 3: Divide by 10^baseTokenDecimals
		// baseDivisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(baseTokenDecimals)), nil)
		// liquidityAmount.Div(liquidityAmount, baseDivisor)

		// // Step 4: Divide by 10^8 (scaling factor)
		// scalingDivisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
		// liquidityAmount.Div(liquidityAmount, scalingDivisor)

		service.marketService.UpdateLiquidity(
			marketTicker,
			order.Size,
			new(big.Int),
		)
	} else {
		// liquidityAmount := new(big.Int)
		// // Step 1: Multiply amount * price
		// liquidityAmount.Mul(order.Amount, order.Price)

		// // Step 2: Multiply by 10^baseTokenDecimals
		// baseMultiplier := new(
		// 	big.Int,
		// ).Exp(big.NewInt(10), big.NewInt(int64(baseTokenDecimals)), nil)
		// liquidityAmount.Mul(liquidityAmount, baseMultiplier)

		// // Step 3: Divide by 10^baseTokenDecimals
		// quoteDivisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(quoteTokenDecimals)), nil)
		// liquidityAmount.Div(liquidityAmount, quoteDivisor)

		// // Step 4: Divide by 10^8 (scaling factor)
		// scalingDivisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
		// liquidityAmount.Div(liquidityAmount, scalingDivisor)
		service.marketService.UpdateLiquidity(
			marketTicker,
			new(big.Int),
			order.Size,
		)
	}
}

func (service *OrderService) FillOrder(order Order, marketTicker string) {
	orderBook := service.OrderBooks[marketTicker]
	amountRemaining := new(big.Int).Set(order.Size)

	if order.OrderType == BuyOrder {
		orderBookWiped := false
		for amountRemaining.Cmp(big.NewInt(0)) > 0 && !orderBookWiped {
			sellIndex := orderBook.SellIndex
			if sellIndex > len(orderBook.Orders) || len(orderBook.Orders) == 0 {
				orderBookWiped = true
				break
			}
			sellOrder := orderBook.Orders[sellIndex]
			amountAvailable := new(big.Int).Sub(sellOrder.Size, sellOrder.SizeFilled)
			if amountAvailable.Cmp(amountRemaining) >= 0 {
				order.SizeFilled.Add(order.SizeFilled, amountRemaining)
				sellOrder.SizeFilled.Add(sellOrder.SizeFilled, amountRemaining)
				amountRemaining = big.NewInt(0)
			} else {
				order.SizeFilled.Add(order.SizeFilled, amountAvailable)
				amountRemaining.Sub(amountRemaining, amountAvailable)
				sellOrder.SizeFilled.Add(sellOrder.SizeFilled, amountAvailable)
			}

			if sellOrder.SizeFilled.Cmp(sellOrder.Size) == 0 {
				sellOrder.Status = Filled
				orderBook.InActiveOrders = append(orderBook.InActiveOrders, sellOrder)
				orderBook.Orders = append(
					orderBook.Orders[:sellIndex],
					orderBook.Orders[sellIndex+1:]...)
			} else {
				orderBook.Orders[sellIndex] = sellOrder
			}
			orderBook.LastPrice = sellOrder.Price
		}
	} else {

	}
	order.Status = Filled
	orderBook.InActiveOrders = append(orderBook.InActiveOrders, order)
	service.OrderBooks[marketTicker] = orderBook
}

func (service *OrderService) GetOrdersByMarketTicker(marketTicker string) []Order {
	return append([]Order{}, service.OrderBooks[marketTicker].Orders...)
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

	for _, order := range orderBook.Orders {
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

	for _, order := range orderBook.InActiveOrders {
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
