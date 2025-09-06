package service

import (
	"fmt"
	"math/big"
)

type MarketService struct {
	Markets       map[string]Market
	MarketTickers []string
}

type Market struct {
	BaseToken                string
	QuoteToken               string
	BaseTokenDecimals        int
	QuoteTokenDecimals       int
	MarketTicker             string
	BuyLiquidityInBaseToken  *big.Int
	SellLiquidityInBaseToken *big.Int
}

func NewMarketService() *MarketService {
	return &MarketService{
		Markets:       make(map[string]Market),
		MarketTickers: []string{},
	}
}

func (service *MarketService) CreateMarket(
	baseToken string,
	quoteToken string,
	baseTokenDecimals int,
	quoteTokenDecimals int,
) Market {
	marketTicker := GetMarketTicker(baseToken, quoteToken)
	service.MarketTickers = append(service.MarketTickers, marketTicker)
	service.Markets[marketTicker] = Market{
		BaseToken:                baseToken,
		QuoteToken:               quoteToken,
		BaseTokenDecimals:        baseTokenDecimals,
		QuoteTokenDecimals:       quoteTokenDecimals,
		MarketTicker:             marketTicker,
		BuyLiquidityInBaseToken:  big.NewInt(0),
		SellLiquidityInBaseToken: big.NewInt(0),
	}

	return service.Markets[marketTicker]
}

func (service *MarketService) UpdateLiquidity(
	marketTicker string,
	buyLiquidityInBaseToken *big.Int,
	sellLiquidityInBaseToken *big.Int,
) {
	market := service.Markets[marketTicker]
	market.BuyLiquidityInBaseToken.Add(market.BuyLiquidityInBaseToken, buyLiquidityInBaseToken)
	market.SellLiquidityInBaseToken.Add(market.SellLiquidityInBaseToken, sellLiquidityInBaseToken)
	service.Markets[marketTicker] = market
}

func (service *MarketService) GetMarket(marketTicker string) Market {
	return service.Markets[marketTicker]
}

func (service *MarketService) PrintMarkets(marketTicker string) {
	for _, market := range service.Markets {
		if market.MarketTicker == marketTicker {
			baseMultiplier := new(
				big.Int,
			).Exp(big.NewInt(10), big.NewInt(int64(market.BaseTokenDecimals)), nil)
			fmt.Printf(
				"Market: %s\nBaseToken: %s\nQuoteToken: %s\nBaseTokenDecimals: %d\nQuoteTokenDecimals: %d\nBuyLiquidityInBaseToken: %d\nSellLiquidityInBaseToken: %d\n",
				market.MarketTicker,
				market.BaseToken,
				market.QuoteToken,
				market.BaseTokenDecimals,
				market.QuoteTokenDecimals,
				new(big.Int).Div(market.BuyLiquidityInBaseToken, baseMultiplier),
				new(big.Int).Div(
					market.SellLiquidityInBaseToken,
					baseMultiplier,
				),
			)
			return
		}
	}
}

func GetMarketTicker(token0, token1 string) string {
	return fmt.Sprintf("%s/%s", token0, token1)
}
