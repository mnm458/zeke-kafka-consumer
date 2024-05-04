package main

import (
	invoice "github.com/mnm458/zeke-kafka-consumer/pkg/invoice"
	app "github.com/mnm458/zeke-kafka-consumer/pkg/server"

	"log"
	"math/big"

	"fmt"
	"os"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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

	// creates a new consumer and subscribes to your topic
	consumer, consumerErr := kafka.NewConsumer(&conf)
	if consumerErr != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", consumerErr)
	}
	// closes the consumer connection
	defer consumer.Close()

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
}
