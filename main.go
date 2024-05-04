package main

import (
	"encoding/binary"
	"log"
	"math/big"

	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const myABIJSON = `[
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "_onramper",
                "type": "address"
            },
            {
                "internalType": "address",
                "name": "_token",
                "type": "address"
            },
            {
                "internalType": "uint256",
                "name": "_amount",
                "type": "uint256"
            },
            {
                "internalType": "int256",
                "name": "_minFiatRate",
                "type": "int256"
            },
            {
                "internalType": "uint64",
                "name": "_dstChainId",
                "type": "uint64"
            }
        ],
        "name": "addOrder",
        "outputs": [
            {
                "internalType": "bytes32",
                "name": "",
                "type": "bytes32"
            }
        ],
        "stateMutability": "nonpayable",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "state",
        "outputs": [
            {
                "internalType": "string",
                "name": "",
                "type": "string"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    }
]`

func ReadConfig() kafka.ConfigMap {
	// reads the client configuration from client.properties
	// and returns it as a key-value map
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

func main() {

	const url = "https://rpc.ankr.com/base_sepolia/5b3e494193a39bb51d06eb3cb4735c305870850adcd0987f03bd6f88ea308737"
	rpcClient, err := ethclient.Dial(url)

	if err != nil {
		panic(err)
	}
	defer rpcClient.Close()
	txHash := common.HexToHash("0xdc9b7a34ca6699872049836692832a1dd54acfe2bef6178d636627d626ff55b3")
	//functionSignature := "addOrder(address, address, uint256, int256 , uint64) bytes32"
	//selector := crypto.Keccak256([]byte(functionSignature))[:4]

	tx, isPending, err := rpcClient.TransactionByHash(context.Background(), txHash)

	if isPending {
		log.Fatal("Pending transaction cannot be processed")
	}

	if err != nil {
		log.Fatal(err)
	}
	inputData := tx.Data()

	if len(inputData) != 164 {
		log.Fatal("Invalid input data length")
	}

	// Decode the input data based on the parameter types
	param1 := common.BytesToAddress(inputData[4:36])
	param2 := common.BytesToAddress(inputData[36:68])
	param3 := new(big.Int).SetBytes(inputData[68:100])
	param4 := new(big.Int).SetBytes(inputData[100:132])
	param5 := binary.BigEndian.Uint64(inputData[132:140])

	fmt.Printf("Onramper addreess: %s\n", param1.Hex())
	fmt.Printf("Stablecoin address: %s\n", param2.Hex())
	fmt.Printf("Amount: %s\n", param3.String())
	fmt.Printf("MinFiatRate: %s\n", param4.String())
	fmt.Printf("Chain ID: %d\n", param5)

	receipt, err := rpcClient.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the transaction was successful
	if receipt.Status == types.ReceiptStatusSuccessful {
		// Assuming the function returns a single value
		if len(receipt.Logs) > 0 {
			// Get the first log entry
			log := receipt.Logs[0]

			// Extract the return value from the log data
			returnValue := common.BytesToHash(log.Data)

			fmt.Printf("Return Value: %s\n", string(returnValue.Hex()))
		} else {
			fmt.Println("No logs found in the transaction receipt")
		}
	} else {
		fmt.Println("Transaction failed")
	}

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
			fmt.Printf("Consumed event from topic %s: key = %-10s value = %s\n",
				*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
		case kafka.Error:
			fmt.Fprintf(os.Stderr, "%% Error: %v\n", ev)
			run = false
		}
	}

	// closes the consumer connection
	consumer.Close()
}
