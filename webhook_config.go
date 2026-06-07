package swissecho

// WebhookConfig holds the secret and method name for a gateway's inbound webhook.
type WebhookConfig struct {
	// Secret is validated against the URL secret parameter. Requests with a wrong
	// secret are rejected with a 401.
	Secret string
	// Handle is the name of the method on the gateway to call when a webhook arrives.
	// Defaults to "Webhook" if not set.
	Handle string
}
