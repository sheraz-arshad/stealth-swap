package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func GenerateRandomAddress() common.Address {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	return crypto.PubkeyToAddress(privateKey.PublicKey)
}
