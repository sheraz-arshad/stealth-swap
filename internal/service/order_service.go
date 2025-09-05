package service

import (
	"fmt"
	"math/big"
	"time"
)

type OrderBook struct {
	Orders         []Order
	LastPrice      *big.Int
	FirstSellIndex int
	FirstBuyIndex  int
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

type Order struct {
	ID int64
	OrderType
	Size      *big.Int
	Price     *big.Int
	CreatedAt time.Time
	Market    Market
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
	for i, existingOrder := range service.OrderBooks[marketTicker].Orders {
		if existingOrder.Price.Cmp(order.Price) > 0 {
			insertIdx = i + 1
		} else {
			break
		}
	}
	orderBook := service.OrderBooks[marketTicker]
	orderBook.Orders = append(orderBook.Orders, Order{}) // extend slice
	copy(orderBook.Orders[insertIdx+1:], orderBook.Orders[insertIdx:])
	orderBook.Orders[insertIdx] = order
	service.OrderBooks[marketTicker] = orderBook

	if len(orderBook.Orders) == 1 {
		orderBook.LastPrice = order.Price
		service.OrderBooks[marketTicker] = orderBook
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
	// lastPrice := service.marketService.GetMarket(marketTicker).LastPrice
	// if order.OrderType == BuyOrder {

	// } else {

	// }
}

func (service *OrderService) GetOrdersByMarketTicker(marketTicker string) []Order {
	return append([]Order{}, service.OrderBooks[marketTicker].Orders...)
}

func (service *OrderService) GetNextOrderID() int64 {
	service.orderID++
	return service.orderID
}

// PrintOrders prints all orders to console in a formatted way
func (service *OrderService) PrintOrders(marketTicker string) {
	fmt.Println("=== ORDERS ===")
	orderBook := service.OrderBooks[marketTicker]
	if len(orderBook.Orders) == 0 {
		fmt.Println("No orders found")
		return
	}

	market := service.marketService.GetMarket(marketTicker)
	quoteMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.QuoteTokenDecimals)), nil)
	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.BaseTokenDecimals)), nil)

	fmt.Println("Last Price:", new(big.Int).Div(orderBook.LastPrice, quoteMultiplier))

	for _, order := range orderBook.Orders {
		fmt.Printf("Order ID:%d | Type: %s | Size: %d | Price: %d | Market: %s | Time: %s\n",
			order.ID,
			order.OrderType,
			new(big.Int).Div(order.Size, baseMultiplier),
			new(big.Int).Div(order.Price, quoteMultiplier),
			marketTicker,
			order.CreatedAt.Format("15:04:05"),
		)
	}
	fmt.Println("=============")
}
