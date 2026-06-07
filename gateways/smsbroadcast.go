package gateways

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	swissecho "github.com/tekkenking/swissecho_go"
)

// SMSBroadcastGateway implements the Swissecho Gateway interface for smsbroadcast.com.au.
// Auth keys required: "username", "password"
// URL: set via GatewayConfig.URL
type SMSBroadcastGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *SMSBroadcastGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *SMSBroadcastGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	username, ok := config.Auth["username"]
	if !ok {
		return nil, fmt.Errorf("smsbroadcast: missing 'username' in Auth config")
	}
	password, ok := config.Auth["password"]
	if !ok {
		return nil, fmt.Errorf("smsbroadcast: missing 'password' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("smsbroadcast: URL is required in GatewayConfig")
	}

	ref := fmt.Sprintf("%d", rand.Int63())
	body := url.Values{}
	body.Set("username", username)
	body.Set("password", password)
	body.Set("to", strings.Join(msg.Recipients, ","))
	body.Set("from", msg.SenderID)
	body.Set("message", msg.Body)
	body.Set("ref", ref)

	resp, err := http.PostForm(config.URL, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("smsbroadcast API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}
