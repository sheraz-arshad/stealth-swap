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