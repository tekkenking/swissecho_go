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
	IdentifierVal interface{} // Optional: tag messages with a user/entity reference for AfterSend tracking
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
// Sender IDs are automatically truncated to 11 characters to comply with SMS standards.
func (m *SwissechoMessage) Sender(sender string) *SwissechoMessage {
	if len(sender) > 11 {
		sender = sender[:11]
	}
	m.SenderID = sender
	return m
}

// Content sets the full body of the message.
// Calling Content() multiple times appends lines, matching PHP behaviour.
func (m *SwissechoMessage) Content(content string) *SwissechoMessage {
	return m.Line(content)
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

// Place sets the geo-routing place key for this message (e.g. "nga", "aus").
func (m *SwissechoMessage) Place(place string) *SwissechoMessage {
	m.PlaceName = place
	return m
}

// Identifier tags this message with an arbitrary reference value (e.g. a user ID).
// The identifier is included in the AfterSend callback for post-send correlation.
func (m *SwissechoMessage) Identifier(id interface{}) *SwissechoMessage {
	m.IdentifierVal = id
	return m
}
