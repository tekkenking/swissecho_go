package swissecho

import (
	"fmt"
	"log"
)

// MockGateway is used when the package is disabled (development mode).
// It logs messages to stdout instead of sending them via an API.
type MockGateway struct {
	config GatewayConfig
	msg    *SwissechoMessage
}

func (m *MockGateway) Boot(config GatewayConfig, msg *SwissechoMessage) error {
	m.config = config
	m.msg = msg
	return nil
}

func (m *MockGateway) Send() (interface{}, error) {
	fakeMode, _ := m.config.Extras["fake"].(string)

	output := fmt.Sprintf("[Swissecho Mock] Mode: %s | Route: %s | To: %v | Sender: %s\nContent: %s",
		fakeMode,
		m.msg.RouteName,
		m.msg.Recipients,
		m.msg.SenderID,
		m.msg.Body,
	)

	if fakeMode == "mail" {
		mailTo, _ := m.config.Extras["fake_mail"].(string)
		log.Printf("MOCK EMAIL -> %s\n%s\n", mailTo, output)
	} else {
		// default to log
		log.Println(output)
	}

	return "mock_success", nil
}
