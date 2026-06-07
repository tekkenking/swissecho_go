package gateways

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	swissecho "github.com/tekkenking/swissecho_go"
)

// WirepickGateway implements the Swissecho Gateway interface for Wirepick SMS.
// Config extras required: "client", "password", "affiliate"
// URL: set via GatewayConfig.URL
type WirepickGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *WirepickGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *WirepickGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	if config.URL == "" {
		return nil, fmt.Errorf("wirepick: URL is required in GatewayConfig")
	}

	client, _ := config.Extras["client"].(string)
	password, _ := config.Extras["password"].(string)
	affiliate, _ := config.Extras["affiliate"].(string)

	if client == "" || password == "" {
		return nil, fmt.Errorf("wirepick: 'client' and 'password' are required in Extras config")
	}

	mobiles := strings.Join(msg.Recipients, ",")
	reqURL := fmt.Sprintf("%s?client=%s&password=%s&affiliate=%s&phone=%s&text=%s&from=%s",
		config.URL,
		url.QueryEscape(client),
		url.QueryEscape(password),
		url.QueryEscape(affiliate),
		url.QueryEscape(mobiles),
		url.QueryEscape(msg.Body),
		url.QueryEscape(msg.SenderID),
	)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wirepick API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
