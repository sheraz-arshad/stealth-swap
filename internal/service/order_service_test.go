package service

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"x-swap/internal/utils"
)

var marketService *MarketService
var marketTicker string
var market Market
var orderService *OrderService
var userService *UserService
var users []common.Address

func setup() {
	marketService = NewMarketService()
	marketService.CreateMarket("BTC", "USD", 8, 6)
	marketTicker = GetMarketTicker("BTC", "USD")
	market = marketService.GetMarket(marketTicker)
	orderService = NewOrderService(marketService)
	userService = NewUserService(orderService, marketService)
	for i := 0; i < 10; i++ {
		users = append(users, utils.GenerateRandomAddress())
		userService.Users[users[i]] = User{
			Balance:       make(map[string]*big.Int),
			BalanceLocked: make(map[string]*big.Int),
		}
	}
}

func topup(user common.Address, amount *big.Int, asset string) {
	if userService.Users[user].Balance[asset] == nil {
		userService.Users[user].Balance[asset] = amount
	} else {
		userService.Users[user].Balance[asset].Add(userService.Users[user].Balance[asset], amount)
	}
	// fmt.Println(userService.Users[user].Balance[asset])
}

func TestCreateBuyOrder(t *testing.T) {
	setup()
	topup(users[0], big.NewInt(200_000e6), "USD")

	orderId := orderService.GetNextOrderID()
	size := big.NewInt(1e8)
	price := big.NewInt(111_000e6)
	order := Order{
		ID:         orderId,
		User:       users[0],
		OrderType:  BuyOrder,
		Size:       size,
		Price:      price,
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	userService.PlaceOrder(order, false)
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
	topup(users[0], big.NewInt(2e8), "BTC")

	orderId := orderService.GetNextOrderID()
	size := big.NewInt(1e8)
	price := big.NewInt(112_000e6)
	order := Order{
		ID:         orderId,
		User:       users[0],
		OrderType:  SellOrder,
		Size:       size,
		Price:      price,
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	userService.PlaceOrder(order, false)
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
	topup(users[0], big.NewInt(500_000e6), "USD")
	// add buy liquidity
	for i := 0; i < 2; i++ {
		order := Order{
			ID:         orderService.GetNextOrderID(),
			User:       users[0],
			OrderType:  BuyOrder,
			Size:       big.NewInt(1e8),
			Price:      big.NewInt(111_000e6 - int64(i*2000e6)),
			SizeFilled: big.NewInt(0),
			CreatedAt:  time.Now(),
			Status:     Open,
			Market:     market,
		}
		userService.PlaceOrder(order, false)
	}

	// fmt.Println(market.BuyLiquidityInBaseToken)
	// fmt.Println(market.SellLiquidityInBaseToken)

	topup(users[1], big.NewInt(5e8), "BTC")
	// add sell liquidity
	for i := 0; i < 2; i++ {
		order := Order{
			ID:         orderService.GetNextOrderID(),
			User:       users[1],
			OrderType:  SellOrder,
			Size:       big.NewInt(1e8),
			Price:      big.NewInt(112_000e6 + int64(i*2000e6)),
			SizeFilled: big.NewInt(0),
			CreatedAt:  time.Now(),
			Status:     Open,
			Market:     market,
		}
		userService.PlaceOrder(order, false)
	}

	// fmt.Println(market.BuyLiquidityInBaseToken)
	// fmt.Println(market.SellLiquidityInBaseToken)

	topup(users[2], big.NewInt(5e8), "BTC")
	// fill order
	order := Order{
		ID:         orderService.GetNextOrderID(),
		User:       users[2],
		OrderType:  SellOrder,
		Size:       big.NewInt(2e8),
		Price:      big.NewInt(112_000e6),
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}
	userService.PlaceOrder(order, true)

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
