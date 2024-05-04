package main

import (
	"log"
	"math/big"

	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

/*
GethParseTx(*ethclient.Client, string) ParsedTx
Parses a transaction and extracts the onRamper address, amount, and intent ID.
It takes an Ethereum client and a transaction hash as input.
Returns a ParsedTx struct containing the extracted information.
*/
func GethParseTx(client *ethclient.Client, txString string) ParsedTx {
	txHash := common.HexToHash(txString)
	tx, isPending, txErr := client.TransactionByHash(context.Background(), txHash)
	if txErr != nil {
		log.Fatal("Transaction retrival failed: ", txErr)
	}
	inputData := tx.Data()

	if len(inputData) != 164 {
		log.Fatal("Invalid input data length")
	}

	if isPending {
		log.Fatal("Pending transaction cannot be processed")
	}
	onRamper := common.BytesToAddress(inputData[4:36]).Hex()
	fmt.Printf("Onramper address: %s\n", onRamper)
	amount := new(big.Int).SetBytes(inputData[68:100])
	fmt.Printf("Amount: %s\n", amount.String())

	parsedTx := ParsedTx{onRamper: onRamper, amount: amount, intentID: GethRetrieveLog(client, txHash)}
	return parsedTx
}

/*
GethRetrieveLog(*ethclient.Client, common.Hash)string
Retrieves the intent ID from the transaction receipt logs.
It takes an Ethereum client and a transaction hash as input.
Returns the intent ID as a string.
*/
func GethRetrieveLog(client *ethclient.Client, txHash common.Hash) string {
	receipt, receiptErr := client.TransactionReceipt(context.Background(), txHash)
	var intentID string
	if receiptErr != nil {
		log.Fatal("Receipt retrieval failed: ", receiptErr)
	}
	// Check if the transaction was successful
	if receipt.Status == types.ReceiptStatusSuccessful {
		if len(receipt.Logs) > 0 {
			// Get the first log entry
			log := receipt.Logs[0]
			// Extract the return value from the log data
			intentID := common.BytesToHash(log.Data).Hex()
			fmt.Printf("Intent ID: %v\n", intentID)
		} else {
			fmt.Println("No intent ID found in the transaction receipt")
		}
	} else {
		log.Fatal("Transaction parsing failed")
	}
	return intentID
}
