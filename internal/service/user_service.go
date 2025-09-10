package service

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type User struct {
	Balance       map[string]*big.Int
	BalanceLocked map[string]*big.Int
}

type UserService struct {
	Users           map[common.Address]User
	UserList        []common.Address
	serviceRegistry *ServiceRegistry
}

func NewUserService() *UserService {
	return &UserService{
		Users:    make(map[common.Address]User),
		UserList: []common.Address{},
	}
}

func (service *UserService) SetServiceRegistry(serviceRegistry *ServiceRegistry) {
	service.serviceRegistry = serviceRegistry
}

func (service *UserService) GetServiceRegistry() (*ServiceRegistry, error) {
	if service.serviceRegistry == nil {
		return nil, errors.New("service registry not set")
	}
	return service.serviceRegistry, nil
}

func (service *UserService) PlaceOrder(order Order, fill bool) {
	// check if order is at the market price, fill it
	// else put it in the order book

	serviceRegistry, err := service.GetServiceRegistry()
	if err != nil {
		panic(err)
	}
	marketService, err := serviceRegistry.GetMarketService()
	if err != nil {
		panic(err)
	}

	market := marketService.GetMarket(order.Market.MarketTicker)
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

	orderService, err := serviceRegistry.GetOrderService()
	if err != nil {
		panic(err)
	}

	if fill {
		orderService.FillOrder(order, order.Market.MarketTicker)
	} else {
		if service.Users[order.User].BalanceLocked[asset] == nil {
			service.Users[order.User].BalanceLocked[asset] = amount
		} else {
			service.Users[order.User].BalanceLocked[asset].Add(service.Users[order.User].BalanceLocked[asset], amount)
		}
		orderService.CreateOrder(order, order.Market.MarketTicker)
	}
}

func (service *UserService) AddBalance(user common.Address, asset string, amount *big.Int) {
	if service.Users[user].Balance[asset] == nil {
		service.Users[user].Balance[asset] = amount
	} else {
		service.Users[user].Balance[asset].Add(service.Users[user].Balance[asset], amount)
	}
}

func (service *UserService) SubBalance(user common.Address, asset string, amount *big.Int) {
	service.Users[user].Balance[asset].Sub(service.Users[user].Balance[asset], amount)
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
