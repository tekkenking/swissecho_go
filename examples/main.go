package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tekkenking/swissecho_go"
	"github.com/tekkenking/swissecho_go/gateways"
)

func main() {
	// 1. Setup Configuration
	config := swissecho.Config{
		Enabled:       false, // Set to false to trigger mock mode automatically
		Fake:          "log",
		DefaultRoute:  "sms",
		DefaultSender: "MyBrand",
		Queue: swissecho.QueueConfig{
			Enabled:      true,
			QueueChannel: "memory", // Use "memory" or "redis"
			Workers:      2,
		},
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				DefaultGateway: "termii",
				Gateways: map[string]swissecho.GatewayConfig{
					"termii": {
						Class:  &gateways.TermiiGateway{},
						Auth:   map[string]string{"api_key": "dummy_api_key"},
						Sender: "TermiiSndr",
					},
				},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "termii", PhoneCode: "234"},
				},
			},
			"whatsapp": {
				DefaultGateway: "termii",
				Gateways: map[string]swissecho.GatewayConfig{
					"termii": {
						Class: &gateways.TermiiGateway{},
						Auth:  map[string]string{"api_key": "dummy_api_key"},
					},
				},
			},
		},
	}

	client := swissecho.New(config)

	// 2. Direct Quick Send
	fmt.Println("--- Quick Send ---")
	_, err := client.Quick("2348012345678", "Your OTP is 9988")
	if err != nil {
		log.Println("Quick Error:", err)
	}

	// 3. Fluent Route Sending
	fmt.Println("\n--- Fluent Route Send ---")
	_, err = client.Route("whatsapp", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348011111111").Content("Hello from WhatsApp!")
	}).Go()
	if err != nil {
		log.Println("Route Error:", err)
	}

	// 4. Async Queue Sending
	fmt.Println("\n--- Async Queue Send ---")
	err = client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348022222222").Content("This is sent via background worker!")
	}).GoAsync()
	if err != nil {
		log.Println("Async Error:", err)
	}

	// 5. Mock Mode Example
	fmt.Println("\n--- Mock Mode Send ---")
	config.Enabled = false // Disable sending globally
	mockClient := swissecho.New(config)
	_, _ = mockClient.Message().To("2348099999999").Content("This should just be logged").Go()

	// Wait briefly to allow async worker to process the message before exiting
	time.Sleep(1 * time.Second)
}
