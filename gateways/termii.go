package gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tekkenking/swissecho_go"
)

// TermiiGateway implements the Swissecho Gateway interface for Termii.
type TermiiGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (t *TermiiGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	t.config = config
	t.msg = msg
	return nil
}

func (t *TermiiGateway) Send() (interface{}, error) {
	// Capture locally to avoid data races if the same struct is reused concurrently
	config := t.config
	msg := t.msg

	apiKey, ok := config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("termii gateway requires 'api_key' in Auth config")
	}

	url := config.URL
	if url == "" {
		url = "https://api.ng.termii.com/api/sms/send"
	}

	channel := "generic"
	if msg.RouteName == "whatsapp" {
		channel = "whatsapp"
	}

	payload := map[string]interface{}{
		"to":      strings.Join(msg.Recipients, ","),
		"from":    msg.SenderID,
		"sms":     msg.Body,
		"type":    "plain",
		"channel": channel,
		"api_key": apiKey,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode termii response: %w", err)
	}

	// Treat non-2xx responses as errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := result["message"].(string)
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("termii API error (HTTP %d): %s", resp.StatusCode, message)
	}

	return result, nil
}
