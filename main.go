package main

import (
		app "github.com/mnm458/zeke-kafka-consumer/pkg/server"

    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

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