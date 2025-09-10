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
var serviceRegistry *ServiceRegistry
var users []common.Address

func setup() {
	marketService = NewMarketService()
	marketService.CreateMarket("BTC", "USD", 8, 6)
	marketTicker = GetMarketTicker("BTC", "USD")
	market = marketService.GetMarket(marketTicker)
	userService = NewUserService()
	orderService = NewOrderService()
	serviceRegistry = NewServiceRegistry(marketService, userService, orderService)
	orderService.SetServiceRegistry(serviceRegistry)
	userService.SetServiceRegistry(serviceRegistry)

	for i := 0; i < 10; i++ {
		users = append(users, utils.GenerateRandomAddress())
		userService.Users[users[i]] = User{
			Balance:       make(map[string]*big.Int),
			BalanceLocked: make(map[string]*big.Int),
		}
	}
}

func topup(user common.Address, amount *big.Int, asset string) {
	userService.AddBalance(user, asset, amount)
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

func TestFillBuyOrder(t *testing.T) {
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

	topup(users[2], big.NewInt(500_000e6), "USD")
	// fill order
	order := Order{
		ID:         orderService.GetNextOrderID(),
		User:       users[2],
		OrderType:  BuyOrder,
		Size:       big.NewInt(2e8),
		Price:      big.NewInt(113_000e6),
		SizeFilled: big.NewInt(0),
		CreatedAt:  time.Now(),
		Status:     Open,
		Market:     market,
	}

	amountIn, amountOut, _ := orderService.GetQuote(order.Clone(), marketTicker)
	user1BalanceBtcBefore := new(big.Int).Set(userService.GetAssetAmount(users[1], "BTC"))
	user1BalanceUsdBefore := new(big.Int).Set(userService.GetAssetAmount(users[1], "USD"))
	user2BalanceBtcBefore := new(big.Int).Set(userService.GetAssetAmount(users[2], "BTC"))
	user2BalanceUsdBefore := new(big.Int).Set(userService.GetAssetAmount(users[2], "USD"))

	userService.PlaceOrder(order, true)

	user1BalanceBtcAfter := new(big.Int).Set(userService.GetAssetAmount(users[1], "BTC"))
	user1BalanceUsdAfter := new(big.Int).Set(userService.GetAssetAmount(users[1], "USD"))
	user2BalanceBtcAfter := new(big.Int).Set(userService.GetAssetAmount(users[2], "BTC"))
	user2BalanceUsdAfter := new(big.Int).Set(userService.GetAssetAmount(users[2], "USD"))

	assert.Equal(
		t,
		user1BalanceBtcBefore.Sub(user1BalanceBtcBefore, user1BalanceBtcAfter),
		amountOut,
	)
	assert.Equal(
		t,
		user1BalanceUsdAfter.Sub(user1BalanceUsdAfter, user1BalanceUsdBefore),
		amountIn,
	)

	assert.Equal(
		t,
		user2BalanceBtcAfter.Sub(user2BalanceBtcAfter, user2BalanceBtcBefore),
		amountOut,
	)
	assert.Equal(
		t,
		user2BalanceUsdBefore.Sub(user2BalanceUsdBefore, user2BalanceUsdAfter),
		amountIn,
	)

	// all sell side is filled
	activeOrders := orderService.GetActiveOrdersByMarketTicker(marketTicker)
	for _, activeOrder := range activeOrders {
		assert.Equal(t, activeOrder.Status, Open)
		assert.Equal(t, activeOrder.SizeFilled, big.NewInt(0))
		assert.Equal(t, activeOrder.OrderType, BuyOrder)
	}

	inActiveOrders := orderService.GetInActiveOrdersByMarketTicker(marketTicker)
	for _, inActiveOrder := range inActiveOrders {
		assert.Equal(t, inActiveOrder.Status, Filled)
		assert.Equal(t, inActiveOrder.SizeFilled, inActiveOrder.Size)
	}

	assert.Equal(t, market.BuyLiquidityInBaseToken, big.NewInt(2e8))
	assert.Equal(t, market.SellLiquidityInBaseToken.String(), big.NewInt(0).String())
}

func TestFillSellOrder(t *testing.T) {
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

	amountIn, amountOut, _ := orderService.GetQuote(order.Clone(), marketTicker)
	user0BalanceBtcBefore := new(big.Int).Set(userService.GetAssetAmount(users[0], "BTC"))
	user0BalanceUsdBefore := new(big.Int).Set(userService.GetAssetAmount(users[0], "USD"))
	user2BalanceBtcBefore := new(big.Int).Set(userService.GetAssetAmount(users[2], "BTC"))
	user2BalanceUsdBefore := new(big.Int).Set(userService.GetAssetAmount(users[2], "USD"))

	userService.PlaceOrder(order, true)

	user0BalanceBtcAfter := new(big.Int).Set(userService.GetAssetAmount(users[0], "BTC"))
	user0BalanceUsdAfter := new(big.Int).Set(userService.GetAssetAmount(users[0], "USD"))
	user2BalanceBtcAfter := new(big.Int).Set(userService.GetAssetAmount(users[2], "BTC"))
	user2BalanceUsdAfter := new(big.Int).Set(userService.GetAssetAmount(users[2], "USD"))

	assert.Equal(t, user0BalanceBtcAfter.Sub(user0BalanceBtcAfter, user0BalanceBtcBefore), amountIn)
	assert.Equal(
		t,
		user0BalanceUsdBefore.Sub(user0BalanceUsdBefore, user0BalanceUsdAfter),
		amountOut,
	)

	assert.Equal(
		t,
		user2BalanceBtcBefore.Sub(user2BalanceBtcBefore, user2BalanceBtcAfter),
		amountIn,
	)
	assert.Equal(
		t,
		user2BalanceUsdAfter.Sub(user2BalanceUsdAfter, user2BalanceUsdBefore),
		amountOut,
	)

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
