package server

import (
	"fmt"
	"log"
	"time"
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type WebhookRequest struct {
	ID           string    `json:"id"`
	CreateTime   time.Time `json:"create_time"`
	ResourceType string    `json:"resource_type"`
	EventType    string    `json:"event_type"`
	Summary      string    `json:"summary"`
	Resource     struct {
		Payments []struct {
			Type            string    `json:"type"`
			TransactionID   string    `json:"transaction_id"`
			TransactionType string    `json:"transaction_type"`
			Method          string    `json:"method"`
		} `json:"payments"`
		TotalAmount struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"total_amount"`
		PaidAmount struct {
			Paypal struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"paypal"`
		} `json:"paid_amount"`
		ID    string `json:"id"`
		Items []struct {
			Name            string `json:"name"`
			Quantity        int    `json:"quantity"`
			UnitPrice       struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"unit_price"`
			UnitOfMeasure string `json:"unit_of_measure"`
		} `json:"items"`
		Status string `json:"status"`
	} `json:"resource"`
}

func Run() {
	errEnv := godotenv.Load()
	if errEnv != nil {
		log.Fatal("Error loading .env file", errEnv)
	}

	redisBaseUrl := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	// Initialise client to connect to db instance 1 for message broking
	client := redis.NewClient(&redis.Options{
		Addr: redisBaseUrl+":"+redisPort,
		Password: "",
		DB: 1,
	})

	defer client.Close()

	// Initialize a new Fiber app
	app := fiber.New()

	// Define a route for the GET method on the root path '/'
	app.Get("/", func(c fiber.Ctx) error {
		// Send a string response to the client
		return c.SendString("Hi 👋!")
	})

	app.Post("/webhook", func(c fiber.Ctx) error {
		var request WebhookRequest
		unmarshalErr := json.Unmarshal([]byte(c.Body()), &request)
		if unmarshalErr != nil {
			log.Fatal("Error:", unmarshalErr)
		}

		invoice_id := request.Resource.ID

		fmt.Printf("inv_id", invoice_id)

		persistErr := client.LPush(context.Background(), "queue", invoice_id).Err()
    if persistErr != nil {
        panic(persistErr)
    }

		// TODO: use for completing order
		// result, readErr := client.BLPop(context.Background(), 0, "queue").Result()
    // if readErr != nil {
    //     log.Fatal("Error reading from queue:", readErr)
    // }		

		return c.SendString("Invoice_id: "+invoice_id+" added to queue ready to be processed.")
	})

// Start the server on port 3000
log.Fatal(app.Listen(":3000"))
}