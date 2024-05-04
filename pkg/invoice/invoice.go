package invoice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
)

type Body struct {
	Rel    string `json:"rel"`
	Href   string `json:"href"`
	Method string `json:"method"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	AppID       string `json:"app_id"`
	ExpiresIn   int    `json:"expires_in"`
}

func CreatePayPalInvoice(txAmount int) (string, error) {
	// PayPal API endpoint for creating invoices
	url := "https://api.sandbox.paypal.com/v2/invoicing/invoices"

	errEnv := godotenv.Load()
	if errEnv != nil {
		log.Fatal("Error loading .env file", errEnv)
	}

	// PayPal Access Token (Replace with your actual access token)
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	accessToken, err := generateAccessToken(clientID, clientSecret)
	if err != nil {
		fmt.Println("Error generating access token:", err)
		return err
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate a random 5-digit number
	invoiceNumber := rand.Intn(90000) + 10000

	// JSON payload for the invoice
	payload := []byte(fmt.Sprintf(`{
		"detail": {
			"invoice_number": "INV-%v",
			"reference": "Reference-123",
			"merchant_info": {
				"email": "merchant@example.com"
			},
			"note": "Thank you for your business!",
			"terms": "Payment due upon receipt",
			"currency_code": "USD"
		},
		"items": [
			{
				"name": "Item 1",
				"quantity": "1",
				"unit_amount": {
					"currency_code": "USD",
					"value": "10.00"
				},
				"tax": {
					"name": "Sales Tax",
					"percent": "%d"
				}
			}
		]
	}`, invoiceNumber, txAmount))

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Send HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error: %s", resp.Status)
	}

	fmt.Println("PayPal invoice created successfully!")
	fmt.Println("Response Body:", string(body))

	var bodyJson Body
	errbody := json.Unmarshal([]byte(body), &bodyJson)
	if errbody != nil {
		log.Fatal("Error parsing JSON:", errbody)
	}
	fmt.Println("body json", bodyJson)
	invoice_id := bodyJson.Href[len(bodyJson.Href)-24:]
	fmt.Println(href)

	return invoice_id, nil
}

func generateAccessToken(clientID, clientSecret string) (string, error) {
	// Construct the basic auth string
	authString := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	// Create HTTP request to PayPal API's OAuth endpoint
	req, err := http.NewRequest("POST", "https://api.sandbox.paypal.com/v1/oauth2/token", strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+authString)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send HTTP request
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %s", resp.Status)
	}

	// Parse response JSON
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}
	fmt.Println(tokenResponse.AccessToken)
	return tokenResponse.AccessToken, nil
}

func main() {
	app := fiber.New()

	// Create PayPal invoice when the server starts
	// if err := createPayPalInvoice(); err != nil {
	// 	fmt.Println("Error creating PayPal invoice:", err)
	// 	return
	// }

	// Fiber server starts without any routes
	app.Listen(":3000")
}
