package gateways

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	swissecho "github.com/tekkenking/swissecho_go"
)

// MontNetsGateway implements the Swissecho Gateway interface for MontNets SMS.
// Auth keys required: "username", "password"
// URL: set via GatewayConfig.URL
type MontNetsGateway struct {
	config swissecho.GatewayConfig
	msg    *swissecho.SwissechoMessage
}

func (g *MontNetsGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	g.config = config
	g.msg = msg
	return nil
}

func (g *MontNetsGateway) Send() (interface{}, error) {
	config := g.config
	msg := g.msg

	username, ok := config.Auth["username"]
	if !ok {
		return nil, fmt.Errorf("montnets: missing 'username' in Auth config")
	}
	password, ok := config.Auth["password"]
	if !ok {
		return nil, fmt.Errorf("montnets: missing 'password' in Auth config")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("montnets: URL is required in GatewayConfig")
	}

	timestamp := time.Now().Format("01021504") // MMDDHHmmss
	payload := map[string]interface{}{
		"userid":    username,
		"pwd":       montNetsEncryptPassword(username, password, timestamp),
		"content":   msg.Body,
		"exno":      msg.SenderID,
		"mobile":    strings.Join(msg.Recipients, ","),
		"timestamp": timestamp,
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
		return nil, fmt.Errorf("montnets: failed to decode response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("montnets API error (HTTP %d)", resp.StatusCode)
	}
	return result, nil
}

// montNetsEncryptPassword computes the MontNets password hash:
// MD5(UPPERCASE(userid) + "00000000" + password + timestamp)
func montNetsEncryptPassword(userid, password, timestamp string) string {
	raw := strings.ToUpper(userid) + "00000000" + password + timestamp
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}
