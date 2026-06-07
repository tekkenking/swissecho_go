package swissecho

// Config represents the root configuration for the Swissecho instance.
type Config struct {
	Enabled       bool
	Fake          string // "log" or "mail"
	FakeMail      string
	DefaultSender string
	DefaultRoute  string
	Routes        map[string]RouteConfig
	Queue         QueueConfig
}

// QueueConfig configures background dispatching
type QueueConfig struct {
	Enabled      bool
	QueueChannel string // "memory" or "redis"
	Workers      int    // Number of concurrent workers (default 5)
	Redis        RedisConfig
}

// RedisConfig configures the redis connection if QueueChannel is "redis"
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// RouteConfig represents configuration for a specific route (e.g. sms, voice).
type RouteConfig struct {
	DefaultGateway string
	Gateways       map[string]GatewayConfig
	Places         map[string]Place
}

// GatewayConfig represents configuration for a specific gateway provider.
type GatewayConfig struct {
	Class  Gateway
	URL    string
	Auth   map[string]string
	Sender string
	Extras map[string]interface{}
}

// Place represents a geographic routing location.
type Place struct {
	Gateway   string
	PhoneCode string
}
