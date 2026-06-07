package gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	swissecho "github.com/tekkenking/swissecho_go"
)

// KudismsWhatsappGateway implements the Swissecho Gateway interface for Kudisms WhatsApp.
// Auth keys required: "api_key"
// URL: set via GatewayConfig.URL
type KudismsWhatsappGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *KudismsWhatsappGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *KudismsWhatsappGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	apiKey, ok := config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("kudisms_whatsapp: missing 'api_key' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("kudisms_whatsapp: URL is required in GatewayConfig")
	}
	if len(msg.Recipients) == 0 {
		return nil, fmt.Errorf("kudisms_whatsapp: no recipients specified")
	}

	// Kudisms uses form-urlencoded
	form := url.Values{}
	form.Set("token", apiKey)
	form.Set("template_code", "2147483647")
	form.Set("recipient", msg.Recipients[0]) // WhatsApp sends to single recipient
	form.Set("parameters", msg.Body)

	resp, err := http.PostForm(config.URL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Try reading as plain text
		return nil, fmt.Errorf("kudisms_whatsapp: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kudisms_whatsapp API error (HTTP %d)", resp.StatusCode)
	}
	return result, nil
}

// TermiiWhatsappGateway reuses TermiiGateway but forces the WhatsApp channel.
// This is convenient if you already have Termii configured and want a dedicated WhatsApp gateway entry.
type TermiiWhatsappGateway struct {
	TermiiGateway
}

func (g *TermiiWhatsappGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	// Force route name so TermiiGateway picks the whatsapp channel
	msg.RouteName = "whatsapp"
	return g.TermiiGateway.Boot(config, msg)
}

// DirectWhatsappGateway sends WhatsApp messages using a username/password API (generic).
// Auth keys required: "username", "password"
// URL: set via GatewayConfig.URL
type DirectWhatsappGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *DirectWhatsappGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *DirectWhatsappGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	username, ok := config.Auth["username"]
	if !ok {
		return nil, fmt.Errorf("direct_whatsapp: missing 'username' in Auth config")
	}
	password, ok := config.Auth["password"]
	if !ok {
		return nil, fmt.Errorf("direct_whatsapp: missing 'password' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("direct_whatsapp: URL is required in GatewayConfig")
	}

	// Build JSON payload
	payload := map[string]interface{}{
		"username": username,
		"password": password,
		"message":  msg.Body,
		"sender":   msg.SenderID,
		"mobiles":  bytes.Join(toByteSlices(msg.Recipients), []byte(",")),
		"verbose":  true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(config.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("direct_whatsapp: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("direct_whatsapp API error (HTTP %d)", resp.StatusCode)
	}
	return result, nil
}

func toByteSlices(strs []string) [][]byte {
	out := make([][]byte, len(strs))
	for i, s := range strs {
		out[i] = []byte(s)
	}
	return out
}
