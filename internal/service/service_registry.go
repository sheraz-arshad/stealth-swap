package service

import "errors"

type ServiceRegistry struct {
	MarketService *MarketService
	UserService   *UserService
	OrderService  *OrderService
}

func NewServiceRegistry(
	marketService *MarketService,
	userService *UserService,
	orderService *OrderService,
) *ServiceRegistry {
	return &ServiceRegistry{
		MarketService: marketService,
		UserService:   userService,
		OrderService:  orderService,
	}
}

func (registry *ServiceRegistry) GetMarketService() (*MarketService, error) {
	if registry.MarketService == nil {
		return nil, errors.New("market service not set")
	}
	return registry.MarketService, nil
}

func (registry *ServiceRegistry) GetUserService() (*UserService, error) {
	if registry.UserService == nil {
		return nil, errors.New("user service not set")
	}
	return registry.UserService, nil
}

func (registry *ServiceRegistry) GetOrderService() (*OrderService, error) {
	if registry.OrderService == nil {
		return nil, errors.New("order service not set")
	}
	return registry.OrderService, nil
}
