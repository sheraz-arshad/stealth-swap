package service

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

func (registry *ServiceRegistry) GetMarketService() *MarketService {
	return registry.MarketService
}

func (registry *ServiceRegistry) GetUserService() *UserService {
	return registry.UserService
}

func (registry *ServiceRegistry) GetOrderService() *OrderService {
	return registry.OrderService
}
