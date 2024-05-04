package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ethereum/go-ethereum/ethclient"
	invoice "github.com/mnm458/zeke-kafka-consumer/pkg/invoice"
)

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

func setupKafkaConsumer(conf kafka.ConfigMap) (*kafka.Consumer, error) {
	consumer, err := kafka.NewConsumer(&conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %v", err)
	}
	return consumer, nil
}

func processMessages(consumer *kafka.Consumer, rpcClient *ethclient.Client) {
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
