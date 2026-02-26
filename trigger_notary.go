package main

import (
	"fmt"
	"os"

	"github.com/nathfavour/auracrab/pkg/notary"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run trigger_notary.go <hex_private_key> <log_data>")
		os.Exit(1)
	}

	hexKey := os.Args[1]
	logData := os.Args[2]

	fmt.Printf("Notarizing: %s\n", logData)
	txHash, err := notary.NotarizeActivity(hexKey, logData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("SUCCESS! On-chain Proof (TX Hash): %s\n", txHash)
	fmt.Printf("View on Explorer: https://opbnb-testnet.bscscan.com/tx/%s\n", txHash)
}
