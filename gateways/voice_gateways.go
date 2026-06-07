package gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	swissecho "github.com/tekkenking/swissecho_go"
)

// TermiiVoiceGateway implements the Swissecho Gateway interface for Termii Voice OTP calls.
// Auth keys required: "api_key"
// URL: defaults to "https://api.ng.termii.com/api/sms/otp/call"
type TermiiVoiceGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *TermiiVoiceGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *TermiiVoiceGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	apiKey, ok := config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("termii_voice: missing 'api_key' in Auth config")
	}

	apiURL := config.URL
	if apiURL == "" {
		apiURL = "https://api.ng.termii.com/api/sms/otp/call"
	}

	if len(msg.Recipients) == 0 {
		return nil, fmt.Errorf("termii_voice: no recipients specified")
	}

	payload := map[string]interface{}{
		"api_key":  apiKey,
		"phone_number": msg.Recipients[0],
		"code":     msg.Body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("termii_voice: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := result["message"].(string)
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("termii_voice API error (HTTP %d): %s", resp.StatusCode, message)
	}
	return result, nil
}

// TextngxyzVoiceGateway implements the Swissecho Gateway interface for textng.xyz Voice OTP.
// Auth keys required: "api_key"
// URL: defaults to "https://api.textng.xyz/voice-otp/"
// Optional Extras: "repeat_times" (int, default 1)
type TextngxyzVoiceGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *TextngxyzVoiceGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *TextngxyzVoiceGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	apiKey, ok := config.Auth["api_key"]
	if !ok {
		return nil, fmt.Errorf("textngxyz_voice: missing 'api_key' in Auth config")
	}

	apiURL := config.URL
	if apiURL == "" {
		apiURL = "https://api.textng.xyz/voice-otp/"
	}

	if len(msg.Recipients) == 0 {
		return nil, fmt.Errorf("textngxyz_voice: no recipients specified")
	}

	repeatTimes := 1
	if rt, ok := config.Extras["repeat_times"].(int); ok && rt > 0 {
		repeatTimes = rt
	}

	form := url.Values{}
	form.Set("key", apiKey)
	form.Set("phone", msg.Recipients[0])
	form.Set("message-opt-code", msg.Body)
	form.Set("otp_repeat", fmt.Sprintf("%d", repeatTimes))
	form.Set("custom_ref", fmt.Sprintf("%d", rand.Int63()))

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("textngxyz_voice: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("textngxyz_voice API error (HTTP %d)", resp.StatusCode)
	}
	return result, nil
}
