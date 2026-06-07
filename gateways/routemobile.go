package gateways

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	swissecho "github.com/tekkenking/swissecho_go"
)

// RouteMobileGateway implements the Swissecho Gateway interface for RouteMobile.
// Auth keys required: "username", "password"
// URL: set via GatewayConfig.URL
type RouteMobileGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *RouteMobileGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *RouteMobileGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	username, ok := config.Auth["username"]
	if !ok {
		return nil, fmt.Errorf("routemobile: missing 'username' in Auth config")
	}
	password, ok := config.Auth["password"]
	if !ok {
		return nil, fmt.Errorf("routemobile: missing 'password' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("routemobile: URL is required in GatewayConfig")
	}

	destination := strings.Join(msg.Recipients, ",")
	reqURL := fmt.Sprintf("%s?username=%s&password=%s&type=0&dlr=0&message=%s&destination=%s&source=%s",
		config.URL,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(msg.Body),
		url.QueryEscape(destination),
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
		return nil, fmt.Errorf("routemobile API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
