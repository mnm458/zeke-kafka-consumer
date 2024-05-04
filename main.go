package main

import (
	app "github.com/mnm458/zeke-kafka-consumer/pkg/server"

	"log"
	"math/big"

	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type ParsedTx struct {
	onRamper string
	amount   *big.Int
	intentID string
}

//const myABIJSON = unmarshal

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

	consumer, consumerErr := setupKafkaConsumer(conf)
	if err != nil {
		log.Fatal("Failed to set up Kafka consumer: ", consumerErr)
	}
	defer consumer.Close()

	consumer.SubscribeTopics([]string{topic}, nil)

	// Rest APi for webhook endpoint
	app.Run()

	processMessages(consumer, rpcClient)

}
