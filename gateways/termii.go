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
	apiKey, ok := t.config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("termii gateway requires 'api_key' in Auth config")
	}

	url := t.config.URL
	if url == "" {
		url = "https://api.ng.termii.com/api/sms/send"
	}

	channel := "generic"
	if t.msg.RouteName == "whatsapp" {
		channel = "whatsapp"
	}

	payload := map[string]interface{}{
		"to":       strings.Join(t.msg.Recipients, ","),
		"from":     t.msg.SenderID,
		"sms":      t.msg.Body,
		"type":     "plain",
		"channel":  channel,
		"api_key":  apiKey,
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

	return result, nil
}
