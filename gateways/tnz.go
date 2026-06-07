package gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	swissecho "github.com/tekkenking/swissecho_go"
)

// TnzGateway implements the Swissecho Gateway interface for TNZ (New Zealand).
// Auth keys required: "api_key"
// URL: set via GatewayConfig.URL
type TnzGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *TnzGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *TnzGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	apiKey, ok := config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("tnz: missing 'api_key' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("tnz: URL is required in GatewayConfig")
	}

	type Recipient struct {
		Recipient string `json:"Recipient"`
	}
	recipients := make([]Recipient, len(msg.Recipients))
	for i, num := range msg.Recipients {
		recipients[i] = Recipient{Recipient: num}
	}

	payload := map[string]interface{}{
		"MessageData": map[string]interface{}{
			"Message":      msg.Body,
			"sender":       msg.SenderID,
			"Destinations": recipients,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("tnz: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tnz API error (HTTP %d)", resp.StatusCode)
	}
	return result, nil
}
