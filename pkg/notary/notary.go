package notary

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	OpBNBTestnetRPC = "https://opbnb-testnet-rpc.bnbchain.org"
	ChainID         = 5611
)

// NotarizeActivity pushes a string of data (e.g., build log hash) to opBNB
func NotarizeActivity(hexKey string, logData string) (string, error) {
	client, err := ethclient.Dial(OpBNBTestnetRPC)
	if err != nil {
		return "", fmt.Errorf("failed to connect to opBNB RPC: %v", err)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Embed our "vibe" or log data into the tx
	data := []byte(fmt.Sprintf("AURACRAB_LOG: %s", logData))

	gasLimit := uint64(21000 + (len(data) * 16)) // Adjust for data size
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %v", err)
	}

	// Transaction to self (fromAddress) with 0 BNB value
	tx := types.NewTransaction(nonce, fromAddress, big.NewInt(0), gasLimit, gasPrice, data)

	chainID := big.NewInt(ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}
