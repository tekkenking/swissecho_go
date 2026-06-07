package swissecho

import "strings"

// SwissechoMessage represents the message being constructed before dispatch.
type SwissechoMessage struct {
	Recipients  []string
	SenderID    string
	Body        string
	RouteName   string
	GatewayName string
	PhoneCode   string
	PlaceName   string
}

// NewMessage creates a new, empty SwissechoMessage.
func NewMessage() *SwissechoMessage {
	return &SwissechoMessage{}
}

// To sets the recipients. Multiple recipients can be separated by a comma.
func (m *SwissechoMessage) To(to string) *SwissechoMessage {
	parts := strings.Split(to, ",")
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			m.Recipients = append(m.Recipients, trimmed)
		}
	}
	return m
}

// Sender overrides the default sender for this message.
func (m *SwissechoMessage) Sender(sender string) *SwissechoMessage {
	m.SenderID = sender
	return m
}

// Content sets the full body of the message.
func (m *SwissechoMessage) Content(content string) *SwissechoMessage {
	m.Body = content
	return m
}

// Line appends a line to the message body.
func (m *SwissechoMessage) Line(line string) *SwissechoMessage {
	if m.Body != "" {
		m.Body += "\n"
	}
	m.Body += line
	return m
}

// Route overrides the default route for this message.
func (m *SwissechoMessage) Route(route string) *SwissechoMessage {
	m.RouteName = route
	return m
}

// Gateway overrides the default gateway for this message.
func (m *SwissechoMessage) Gateway(gateway string) *SwissechoMessage {
	m.GatewayName = gateway
	return m
}
