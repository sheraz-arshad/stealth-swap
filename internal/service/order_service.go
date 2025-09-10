package service

import (
	"errors"
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
	OrderBooks      map[string]OrderBook
	serviceRegistry *ServiceRegistry
	orderID         int64
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

func NewOrderService() *OrderService {
	return &OrderService{
		OrderBooks: make(map[string]OrderBook),
		orderID:    0,
	}
}

func (service *OrderService) SetServiceRegistry(serviceRegistry *ServiceRegistry) {
	service.serviceRegistry = serviceRegistry
}

func (service *OrderService) GetServiceRegistry() (*ServiceRegistry, error) {
	if service.serviceRegistry == nil {
		return nil, errors.New("service registry not set")
	}
	return service.serviceRegistry, nil
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

	serviceRegistry, err := service.GetServiceRegistry()
	if err != nil {
		panic(err)
	}
	marketService, err := serviceRegistry.GetMarketService()
	if err != nil {
		panic(err)
	}

	if order.OrderType == BuyOrder {
		marketService.UpdateLiquidity(
			marketTicker,
			new(big.Int).Set(order.Size),
			big.NewInt(0),
		)
	} else {
		marketService.UpdateLiquidity(
			marketTicker,
			big.NewInt(0),
			new(big.Int).Set(order.Size),
		)
	}
}

func (service *OrderService) FillOrder(order Order, marketTicker string) {
	serviceRegistry, err := service.GetServiceRegistry()
	if err != nil {
		panic(err)
	}

	orderBook := service.OrderBooks[marketTicker]
	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(order.Market.BaseTokenDecimals)), nil)
	amountRemaining := new(big.Int).Set(order.Size)

	var takerAmount *big.Int
	if order.OrderType == BuyOrder {
		takerAmount = new(big.Int).Mul(order.Size, order.Price)
		takerAmount.Div(takerAmount, baseMultiplier)
	} else {
		takerAmount = new(big.Int).Set(order.Size)
	}

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
		var fillableAmount *big.Int
		if amountAvailable.Cmp(amountRemaining) >= 0 {
			fillableAmount = amountRemaining
		} else {
			fillableAmount = amountAvailable
		}
		sizeFilled := new(big.Int).Set(fillableAmount)

		userService, err := serviceRegistry.GetUserService()
		if err != nil {
			panic(err)
		}

		if order.OrderType == BuyOrder {
			quoteTokenAmountForMaker := new(big.Int).Mul(fillableAmount, makerOrder.Price)
			quoteTokenAmountForMaker.Div(quoteTokenAmountForMaker, baseMultiplier)

			if quoteTokenAmountForMaker.Cmp(takerAmount) > 0 {
				quoteTokenAmountForMaker = takerAmount
				takerAmount = big.NewInt(0)
				sizeFilled = new(big.Int).Mul(quoteTokenAmountForMaker, baseMultiplier)
				sizeFilled.Div(sizeFilled, makerOrder.Price)
			} else {
				takerAmount.Sub(takerAmount, quoteTokenAmountForMaker)
			}
			// add quote token amount for maker
			userService.AddBalance(
				makerOrder.User,
				order.Market.QuoteToken,
				quoteTokenAmountForMaker,
			)
			// add base token amount (size filled) for taker
			userService.AddBalance(
				order.User,
				order.Market.BaseToken,
				sizeFilled,
			)
			userService.SubBalance(
				makerOrder.User,
				order.Market.BaseToken,
				sizeFilled,
			)
			userService.SubBalance(
				order.User,
				order.Market.QuoteToken,
				quoteTokenAmountForMaker,
			)
		} else {
			if fillableAmount.Cmp(takerAmount) > 0 {
				fillableAmount = takerAmount
				takerAmount = big.NewInt(0)
				sizeFilled = fillableAmount
			} else {
				takerAmount.Sub(takerAmount, fillableAmount)
			}

			quoteTokenAmountForTaker := new(big.Int).Mul(fillableAmount, makerOrder.Price)
			quoteTokenAmountForTaker.Div(quoteTokenAmountForTaker, baseMultiplier)

			// add base token amount (size filled) for maker
			userService.AddBalance(
				makerOrder.User,
				order.Market.BaseToken,
				sizeFilled,
			)
			// add quote token amount for taker
			userService.AddBalance(
				order.User,
				order.Market.QuoteToken,
				quoteTokenAmountForTaker,
			)
			userService.SubBalance(
				order.User,
				order.Market.BaseToken,
				sizeFilled,
			)
			userService.SubBalance(
				makerOrder.User,
				order.Market.QuoteToken,
				quoteTokenAmountForTaker,
			)
		}
		amountRemaining.Sub(amountRemaining, sizeFilled)
		order.SizeFilled.Add(order.SizeFilled, sizeFilled)
		makerOrder.SizeFilled.Add(makerOrder.SizeFilled, sizeFilled)

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

	marketService, err := serviceRegistry.GetMarketService()
	if err != nil {
		panic(err)
	}

	if order.OrderType == BuyOrder {
		marketService.UpdateLiquidity(
			marketTicker,
			big.NewInt(0),
			new(big.Int).Neg(order.SizeFilled),
		)
	} else {
		marketService.UpdateLiquidity(
			marketTicker,
			new(big.Int).Neg(order.SizeFilled),
			big.NewInt(0),
		)
	}
}

func (service *OrderService) GetQuote(
	order Order,
	marketTicker string,
) (*big.Int, *big.Int, *big.Int) {
	orderBook := service.OrderBooks[marketTicker]
	orders := make([]Order, len(orderBook.Orders))
	for i, order := range orderBook.Orders {
		orders[i] = order.Clone()
	}
	buyIndex := orderBook.BuyIndex
	sellIndex := orderBook.SellIndex
	amountRemaining := new(big.Int).Set(order.Size)
	amountOut := big.NewInt(0)

	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(order.Market.BaseTokenDecimals)), nil)

	var takerAmount *big.Int
	if order.OrderType == BuyOrder {
		takerAmount = new(big.Int).Mul(order.Size, order.Price)
		takerAmount.Div(takerAmount, baseMultiplier)
	} else {
		takerAmount = new(big.Int).Set(order.Size)
	}
	_takerAmount := new(big.Int).Set(takerAmount)

	for amountRemaining.Cmp(big.NewInt(0)) > 0 {
		if len(orderBook.Orders) == 0 {
			break
		}
		var makerIndex int
		if order.OrderType == BuyOrder {
			makerIndex = sellIndex
		} else {
			makerIndex = buyIndex
		}

		if makerIndex < 0 || makerIndex >= len(orders) {
			break
		}

		makerOrder := orders[makerIndex]
		amountAvailable := new(big.Int).Sub(makerOrder.Size, makerOrder.SizeFilled)
		var fillableAmount *big.Int
		if amountAvailable.Cmp(amountRemaining) >= 0 {
			fillableAmount = amountRemaining
		} else {
			fillableAmount = amountAvailable
		}
		sizeFilled := new(big.Int).Set(fillableAmount)

		if order.OrderType == BuyOrder {
			quoteTokenAmountForMaker := new(big.Int).Mul(fillableAmount, makerOrder.Price)
			quoteTokenAmountForMaker.Div(quoteTokenAmountForMaker, baseMultiplier)

			if quoteTokenAmountForMaker.Cmp(takerAmount) > 0 {
				quoteTokenAmountForMaker = takerAmount
				takerAmount = big.NewInt(0)
				sizeFilled = new(big.Int).Mul(quoteTokenAmountForMaker, baseMultiplier)
				sizeFilled.Div(sizeFilled, makerOrder.Price)
			} else {
				takerAmount.Sub(takerAmount, quoteTokenAmountForMaker)
			}
			amountOut.Add(amountOut, sizeFilled)
		} else {
			if fillableAmount.Cmp(takerAmount) > 0 {
				fillableAmount = takerAmount
				takerAmount = big.NewInt(0)
				sizeFilled = fillableAmount
			} else {
				takerAmount.Sub(takerAmount, fillableAmount)
			}

			quoteTokenAmountForTaker := new(big.Int).Mul(fillableAmount, makerOrder.Price)
			quoteTokenAmountForTaker.Div(quoteTokenAmountForTaker, baseMultiplier)

			amountOut.Add(amountOut, quoteTokenAmountForTaker)
		}
		amountRemaining.Sub(amountRemaining, sizeFilled)
		order.SizeFilled.Add(order.SizeFilled, sizeFilled)
		makerOrder.SizeFilled.Add(makerOrder.SizeFilled, sizeFilled)

		if makerOrder.SizeFilled.Cmp(makerOrder.Size) == 0 {
			orders = append(
				orders[:makerIndex],
				orders[makerIndex+1:]...)

			if order.OrderType == SellOrder {
				sellIndex = sellIndex - 1
				buyIndex = buyIndex - 1
			}
		} else {
			orders[makerIndex] = makerOrder
		}
	}

	amountIn := new(big.Int).Sub(_takerAmount, takerAmount)
	var executionPrice *big.Int
	if order.OrderType == BuyOrder {
		executionPrice = new(big.Int).Mul(amountIn, baseMultiplier)
		executionPrice.Div(executionPrice, amountOut)
	} else {
		executionPrice = new(big.Int).Mul(amountOut, baseMultiplier)
		executionPrice.Div(executionPrice, amountIn)
	}

	return amountIn, amountOut, executionPrice
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

func (order Order) Clone() Order {
	return Order{
		ID:         order.ID,
		User:       order.User,
		OrderType:  order.OrderType,
		Size:       new(big.Int).Set(order.Size),
		Price:      new(big.Int).Set(order.Price),
		SizeFilled: new(big.Int).Set(order.SizeFilled),
		CreatedAt:  order.CreatedAt,
		Status:     order.Status,
		Market:     order.Market,
	}
}

// PrintOrders prints all orders to console in a formatted way
func (service *OrderService) PrintActiveOrders(marketTicker string) {
	fmt.Println("=== ACTIVE ORDERS ===")

	serviceRegistry, err := service.GetServiceRegistry()
	if err != nil {
		panic(err)
	}
	marketService, err := serviceRegistry.GetMarketService()
	if err != nil {
		panic(err)
	}
	market := marketService.GetMarket(marketTicker)
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

	serviceRegistry, err := service.GetServiceRegistry()
	if err != nil {
		panic(err)
	}
	marketService, err := serviceRegistry.GetMarketService()
	if err != nil {
		panic(err)
	}
	market := marketService.GetMarket(marketTicker)
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
