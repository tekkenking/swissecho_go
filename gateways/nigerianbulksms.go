package gateways

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	swissecho "github.com/tekkenking/swissecho_go"
)

// NigerianBulkSMSGateway implements the Swissecho Gateway interface for nigeriabulksms.com.
// Auth keys required: "username", "password"
// URL: defaults to "https://portal.nigeriabulksms.com/api/"
type NigerianBulkSMSGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *NigerianBulkSMSGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *NigerianBulkSMSGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	username, ok := config.Auth["username"]
	if !ok {
		return nil, fmt.Errorf("nigerianbulksms: missing 'username' in Auth config")
	}
	password, ok := config.Auth["password"]
	if !ok {
		return nil, fmt.Errorf("nigerianbulksms: missing 'password' in Auth config")
	}

	apiURL := config.URL
	if apiURL == "" {
		apiURL = "https://portal.nigeriabulksms.com/api/"
	}

	mobiles := strings.Join(msg.Recipients, ",")
	reqURL := fmt.Sprintf("%s?username=%s&password=%s&message=%s&sender=%s&mobiles=%s&verbose=true",
		apiURL,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(msg.Body),
		url.QueryEscape(msg.SenderID),
		url.QueryEscape(mobiles),
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
		return nil, fmt.Errorf("nigerianbulksms API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
