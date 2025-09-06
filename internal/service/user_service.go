package service

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type User struct {
	Balance       map[string]*big.Int
	BalanceLocked map[string]*big.Int
}

type UserService struct {
	Users         map[common.Address]User
	UserList      []common.Address
	orderService  *OrderService
	marketService *MarketService
}

func NewUserService(orderService *OrderService, marketService *MarketService) *UserService {
	return &UserService{
		Users:         make(map[common.Address]User),
		UserList:      []common.Address{},
		orderService:  orderService,
		marketService: marketService,
	}
}

func (service *UserService) PlaceOrder(order Order, fill bool) {
	// check if order is at the market price, fill it
	// else put it in the order book

	market := service.marketService.GetMarket(order.Market.MarketTicker)
	baseMultiplier := new(
		big.Int,
	).Exp(big.NewInt(10), big.NewInt(int64(market.BaseTokenDecimals)), nil)

	var asset string
	var amount *big.Int
	if order.OrderType == BuyOrder {
		asset = market.QuoteToken
		amount = new(big.Int).Mul(order.Size, order.Price)
		amount.Div(amount, baseMultiplier)
	} else {
		asset = market.BaseToken
		amount = order.Size
	}

	assetBalance := service.Users[order.User].Balance[asset]
	if assetBalance.Cmp(amount) < 0 {
		panic("Insufficient balance")
	}

	if service.Users[order.User].BalanceLocked[asset] == nil {
		service.Users[order.User].BalanceLocked[asset] = amount
	} else {
		service.Users[order.User].BalanceLocked[asset].Add(service.Users[order.User].BalanceLocked[asset], amount)
	}

	if fill {
		service.orderService.FillOrder(order, order.Market.MarketTicker)
	} else {
		service.orderService.CreateOrder(order, order.Market.MarketTicker)
	}
}

func (service *UserService) GetAssetAmount(user common.Address, asset string) *big.Int {
	return service.Users[user].Balance[asset]
}

func (service *UserService) GetAssetAmountLocked(user common.Address, asset string) *big.Int {
	return service.Users[user].BalanceLocked[asset]
}

func (service *UserService) GetAssetAmountAvailable(user common.Address, asset string) *big.Int {
	return new(big.Int).Sub(
		service.Users[user].Balance[asset],
		service.Users[user].BalanceLocked[asset],
	)
}
