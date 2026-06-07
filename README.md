# Swissecho Go

Swissecho Go is a fluent, multi-channel (SMS, Voice, WhatsApp, Slack), multi-gateway messaging package for Go, inspired by the popular Laravel Swissecho package.

## Features
- **Multi-channel & Multi-gateway:** Easily switch between SMS, Voice, and WhatsApp using different providers.
- **Fluent API:** A clean, chainable builder pattern to construct your messages.
- **Geo-Routing (Places):** Automatically select the best gateway and prepend correct country codes based on "places" mapping.
- **Mock Mode:** Log messages or "send" them via email in your development environment instead of hitting real APIs.
- **Extensible:** Adding your own gateway is as simple as implementing a 2-method interface.

## Installation

```bash
go get github.com/tekkenking/swissecho_go
```

## Basic Usage

### Configuration

You configure Swissecho by creating a `swissecho.Config` object, injecting the gateways you want to use.

```go
package main

import (
	"github.com/tekkenking/swissecho_go"
	"github.com/tekkenking/swissecho_go/gateways"
)

func main() {
	config := swissecho.Config{
		Enabled:       true, // Set to false to trigger mock mode
		DefaultRoute:  "sms",
		DefaultSender: "MyApp",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				DefaultGateway: "termii",
				Gateways: map[string]swissecho.GatewayConfig{
					"termii": {
						Class: &gateways.TermiiGateway{},
						Auth:  map[string]string{"api_key": "your_api_key"},
					},
				},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "termii", PhoneCode: "234"},
				},
			},
		},
	}

	client := swissecho.New(config)
    
    // Quick send
    client.Quick("2348012345678", "Your OTP is 1234")
}
```

### Fluent API
You can build complex messages using the fluent API:

```go
client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.To("2348012345678, 2348098765432").
        Content("Your order has been shipped!").
        Line("Track it at https://example.com/track")
}).Go()
```

### Custom Gateways
To build your own gateway, just implement the `swissecho.Gateway` interface:

```go
type Gateway interface {
	Boot(config GatewayConfig, msg *SwissechoMessage) error
	Send() (interface{}, error)
}
```
And register it in your config `Routes` block!
