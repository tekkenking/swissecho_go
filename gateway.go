package swissecho

// Gateway defines the interface that all provider implementations must satisfy.
type Gateway interface {
	Boot(config GatewayConfig, msg *SwissechoMessage) error
	Send() (interface{}, error)
}
