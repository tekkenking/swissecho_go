package swissecho

import "net/http"

// Gateway defines the interface that all provider implementations must satisfy.
type Gateway interface {
	Boot(config GatewayConfig, msg *SwissechoMessage) error
	Send() (interface{}, error)
}

// WebhookReceiver is an optional interface a Gateway can implement to handle
// inbound webhook callbacks from its provider (e.g. delivery receipts).
// Register it with webhook.NewHandler(config) to receive calls.
type WebhookReceiver interface {
	HandleWebhook(r *http.Request) error
}

