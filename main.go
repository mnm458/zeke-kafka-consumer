package main

import (
	invoice "github.com/mnm458/zeke-kafka-consumer/pkg/invoice"
	app "github.com/mnm458/zeke-kafka-consumer/pkg/server"

	"log"
	"math/big"

	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type ParsedTx struct {
	onRamper string
	amount   *big.Int
	intentID string
}

//const myABIJSON = unmarshal

/*
ReadConfig()
Reads the client configuration from client.properties
Returns it as a key-value map
*/
func ReadConfig() kafka.ConfigMap {
	//
	m := make(map[string]kafka.ConfigValue)
	file, err := os.Open("client.properties")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %s", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") && len(line) != 0 {
			kv := strings.Split(line, "=")
			parameter := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			m[parameter] = value
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Failed to read file: %s", err)
		os.Exit(1)
	}

	return m
}

/*
GethParseTx()
Reads the client configuration from client.properties
Returns it as a key-value map
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

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	apiKey := os.Getenv("ANKR_API_KEY")
	url := fmt.Sprintf("https://rpc.ankr.com/base_sepolia/%s", apiKey)
	rpcClient, rpcErr := ethclient.Dial(url)

	if rpcErr != nil {
		panic(rpcErr)
	}
	defer rpcClient.Close()

	//Kafka config
	topic := "orders_topic"
	conf := ReadConfig()

	// sets the consumer group ID and offset
	conf["group.id"] = "go-group-1"
	conf["auto.offset.reset"] = "earliest"

	// creates a new consumer and subscribes to your topic
	consumer, _ := kafka.NewConsumer(&conf)
	consumer.SubscribeTopics([]string{topic}, nil)

	// Rest APi for webhook endpoint
	app.Run()

	run := true
	for run {
		// consumes messages from the subscribed topic and prints them to the console
		e := consumer.Poll(1000)
		switch ev := e.(type) {
		case *kafka.Message:
			// application-specific processing
			// TODO: use go fiber client to make invoice requests
			// TODO: persist key pair - invoice_id, order_id
			fmt.Printf("Consumed event from topic %s: key = %-10s value = %s\n",
				*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
			parsedTx := GethParseTx(rpcClient, string(ev.Value))
			intentIdInt, IntErr := strconv.Atoi(parsedTx.intentID)
			if IntErr != nil {
				log.Fatal("Corrupted intent ID")
			}
			invoice_id, _ := invoice.CreatePayPalInvoice(intentIdInt)
			//send the txAmount to create PayPal invoice
			fmt.Println(invoice_id)
		case kafka.Error:
			fmt.Fprintf(os.Stderr, "%% Error: %v\n", ev)
			run = false
		}
	}

	// closes the consumer connection
	consumer.Close()
}
