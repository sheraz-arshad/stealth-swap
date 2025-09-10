package service

import (
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockchainService struct {
	client             *ethclient.Client
	serviceRegistry    *ServiceRegistry
	ctx                context.Context
	cancel             context.CancelFunc
	rpcURL             string
	pollInterval       time.Duration
	contractAddress    common.Address
	lastProcessedBlock uint64
}

type TransferEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Token  string
	Block  uint64
	TxHash string
}

func NewBlockchainService(rpcURL string, contractAddress common.Address) *BlockchainService {
	ctx, cancel := context.WithCancel(context.Background())

	return &BlockchainService{
		rpcURL:             rpcURL,
		contractAddress:    contractAddress,
		pollInterval:       5 * time.Second,
		ctx:                ctx,
		cancel:             cancel,
		lastProcessedBlock: 0,
	}
}

func (service *BlockchainService) SetServiceRegistry(serviceRegistry *ServiceRegistry) {
	service.serviceRegistry = serviceRegistry
}

func (service *BlockchainService) GetServiceRegistry() (*ServiceRegistry, error) {
	if service.serviceRegistry == nil {
		return nil, errors.New("service registry not set")
	}
	return service.serviceRegistry, nil
}

func (service *BlockchainService) Start() error {
	log.Println("Starting blockchain service...")
	client, err := ethclient.Dial(service.rpcURL)
	if err != nil {
		return err
	}
	service.client = client

	latestBlock, err := service.client.BlockNumber(service.ctx)
	if err != nil {
		return err
	}

	if latestBlock > 10 {
		service.lastProcessedBlock = latestBlock - 10
	} else {
		service.lastProcessedBlock = 0
	}
	service.eventPollingLoop()

	return nil
}

func (service *BlockchainService) Stop() {
	log.Println("Stopping blockchain service...")
	service.cancel()
	if service.client != nil {
		service.client.Close()
	}
}

func (service *BlockchainService) eventPollingLoop() {
	ticker := time.NewTicker(service.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-service.ctx.Done():
			log.Println("Blockchain service polling stopped")
			return
		case <-ticker.C:
			if err := service.processNewBlocks(); err != nil {
				log.Printf("Error processing blocks: %v", err)
			}
		}
	}
}

func (service *BlockchainService) processNewBlocks() error {
	latestBlock, err := service.client.BlockNumber(service.ctx)
	if err != nil {
		return err
	}

	if latestBlock <= service.lastProcessedBlock {
		return nil
	}

	log.Printf("Processing blocks %d to %d", service.lastProcessedBlock+1, latestBlock)
	for blockNum := service.lastProcessedBlock + 1; blockNum <= latestBlock; blockNum++ {
		if err := service.processBlock(blockNum); err != nil {
			log.Printf("Error processing block %d: %v", blockNum, err)
			continue
		}
	}

	service.lastProcessedBlock = latestBlock
	return nil
}

func (service *BlockchainService) processBlock(blockNumber uint64) error {
	block, err := service.client.BlockByNumber(service.ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return err
	}

	for _, tx := range block.Transactions() {
		if service.isRelevantTransaction(tx) {
			transferEvent, err := service.parseTransferEvent(tx, blockNumber)
			if err != nil {
				log.Printf("Error parsing transfer event: %v", err)
				continue
			}

			if transferEvent != nil {
				if err := service.updateUserBalance(transferEvent); err != nil {
					log.Printf("Error updating user balance: %v", err)
				}
			}
		}
	}

	return nil
}

func (service *BlockchainService) isRelevantTransaction(tx *types.Transaction) bool {
	// to is valid token address

	if tx.To() != nil && *tx.To() == service.contractAddress {
		return true
	}
	return false
}

func (service *BlockchainService) parseTransferEvent(
	tx *types.Transaction,
	blockNumber uint64,
) (*TransferEvent, error) {
	transferEventSignature := []byte("Transfer(address,address,uint256)")
	transferEventSigHash := common.BytesToHash(
		crypto.Keccak256(transferEventSignature),
	)

	receipt, err := service.client.TransactionReceipt(service.ctx, tx.Hash())
	if err != nil {
		return nil, err
	}

	for _, vLog := range receipt.Logs {
		if len(vLog.Topics) == 0 || vLog.Topics[0] != transferEventSigHash {
			continue
		}
		from := common.BytesToAddress(vLog.Topics[1].Bytes())
		to := common.BytesToAddress(vLog.Topics[2].Bytes())
		amount := new(big.Int).SetBytes(vLog.Data)

		return &TransferEvent{
			From:   from,
			To:     to,
			Amount: amount,
			Token:  tx.To().Hex(),
			Block:  blockNumber,
			TxHash: tx.Hash().Hex(),
		}, nil
	}

	return nil, nil
}

// updateUserBalance updates user balances based on transfer events
func (service *BlockchainService) updateUserBalance(event *TransferEvent) error {
	// serviceRegistry, err := service.GetServiceRegistry()
	// if err != nil {
	// 	return err
	// }

	// userService, err := serviceRegistry.GetUserService()
	// if err != nil {
	// 	return err
	// }

	log.Printf("Processing transfer: %s -> %s, Amount: %s %s",
		event.From.Hex(), event.To.Hex(), event.Amount.String(), event.Token)

	// Add balance to recipient
	// userService.AddBalance(event.To, event.Token, event.Amount)
	return nil
}

// GetLastProcessedBlock returns the last processed block number
func (service *BlockchainService) GetLastProcessedBlock() uint64 {
	return service.lastProcessedBlock
}

// SetPollInterval allows changing the polling interval
func (service *BlockchainService) SetPollInterval(interval time.Duration) {
	service.pollInterval = interval
}
