# Swissecho Go

A fluent, multi-channel messaging package for Go. Send SMS, WhatsApp, Voice, and Slack messages through multiple providers using a clean, chainable API — with built-in support for background queueing, geo-routing, and a developer-friendly mock mode.

> Inspired by the Laravel [swissecho](https://github.com/tekkenking/swissecho) package.

---

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
  - [Config](#config)
  - [QueueConfig](#queueconfig)
  - [RedisConfig](#redisconfig)
  - [RouteConfig](#routeconfig)
  - [GatewayConfig](#gatewayconfig)
  - [Place (Geo-Routing)](#place-geo-routing)
- [Sending Messages](#sending-messages)
  - [Quick Send](#quick-send)
  - [Fluent Builder API](#fluent-builder-api)
  - [Async Sending](#async-sending)
- [Routing & Channels](#routing--channels)
- [Geo-Routing with Places](#geo-routing-with-places)
- [Mock / Development Mode](#mock--development-mode)
- [Built-in Queue](#built-in-queue)
  - [Memory Queue](#memory-queue)
  - [Redis Queue](#redis-queue)
- [Building a Custom Gateway](#building-a-custom-gateway)
- [Included Gateways](#included-gateways)
  - [Termii](#termii)
- [Error Handling](#error-handling)

---

## Features

- **Multi-channel:** SMS, WhatsApp, Voice, Slack — send via any channel by name.
- **Multi-gateway:** Plug in multiple providers per channel and switch between them.
- **Fluent API:** Chainable builder pattern — readable and concise message construction.
- **Async Queueing:** Built-in background worker pool using Go channels (memory) or Redis.
- **Geo-Routing:** Automatically route messages to the right gateway and prepend country codes based on region.
- **Mock Mode:** In development, log messages to stdout or fake-email them instead of calling real APIs.
- **Extensible:** Add your own gateway by implementing a simple 2-method interface.

---

## Installation

```bash
go get github.com/tekkenking/swissecho_go
```

> **Note:** If you plan to use the Redis queue, the `github.com/redis/go-redis/v9` dependency is already bundled with this package. No extra install needed.

---

## Quick Start

```go
package main

import (
    "log"

    "github.com/tekkenking/swissecho_go"
    "github.com/tekkenking/swissecho_go/gateways"
)

func main() {
    config := swissecho.Config{
        Enabled:       true,
        DefaultRoute:  "sms",
        DefaultSender: "MyApp",
        Routes: map[string]swissecho.RouteConfig{
            "sms": {
                DefaultGateway: "termii",
                Gateways: map[string]swissecho.GatewayConfig{
                    "termii": {
                        Class: &gateways.TermiiGateway{},
                        Auth:  map[string]string{"api_key": "YOUR_API_KEY"},
                    },
                },
            },
        },
    }

    client := swissecho.New(config)

    _, err := client.Quick("2348012345678", "Hello from Swissecho!")
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## Configuration Reference

### `Config`

The root configuration struct passed to `swissecho.New()`.

| Field           | Type                          | Description                                                                 |
|-----------------|-------------------------------|-----------------------------------------------------------------------------|
| `Enabled`       | `bool`                        | Set to `true` to send real messages. Set to `false` to enable Mock Mode.    |
| `Fake`          | `string`                      | Mock mode behaviour: `"log"` (stdout) or `"mail"` (send to a fake email).   |
| `FakeMail`      | `string`                      | The email address to use when `Fake` is `"mail"`.                           |
| `DefaultSender` | `string`                      | The sender ID/name used when a message doesn't specify one.                  |
| `DefaultRoute`  | `string`                      | The route used when a message doesn't specify one (e.g. `"sms"`).           |
| `Routes`        | `map[string]RouteConfig`      | A map of route names to their configuration. See [RouteConfig](#routeconfig).|
| `Queue`         | `QueueConfig`                 | Background queue settings. See [QueueConfig](#queueconfig).                  |

---

### `QueueConfig`

Controls the built-in background message queue.

| Field          | Type          | Description                                                                     |
|----------------|---------------|---------------------------------------------------------------------------------|
| `Enabled`      | `bool`        | Set to `true` to enable the background queue.                                   |
| `QueueChannel` | `string`      | The queue backend: `"memory"` (default, uses Go channels) or `"redis"`.         |
| `Workers`      | `int`         | Number of concurrent goroutines consuming from the queue. Defaults to `5`.      |
| `Redis`        | `RedisConfig` | Redis connection config. Only needed when `QueueChannel` is `"redis"`.          |

---

### `RedisConfig`

Redis connection settings, only relevant when `QueueConfig.QueueChannel = "redis"`.

| Field      | Type     | Description                                      |
|------------|----------|--------------------------------------------------|
| `Addr`     | `string` | Redis address, e.g. `"localhost:6379"`.          |
| `Password` | `string` | Redis password. Leave empty if not set.          |
| `DB`       | `int`    | Redis database number. Defaults to `0`.          |

---

### `RouteConfig`

Configuration for a single messaging channel (e.g. `"sms"`, `"whatsapp"`).

| Field            | Type                          | Description                                                             |
|------------------|-------------------------------|-------------------------------------------------------------------------|
| `DefaultGateway` | `string`                      | The gateway to use by default for this route.                           |
| `Gateways`       | `map[string]GatewayConfig`    | Named gateways available on this route. See [GatewayConfig](#gatewayconfig). |
| `Places`         | `map[string]Place`            | Geo-routing rules. See [Place](#place-geo-routing).                     |

---

### `GatewayConfig`

Configuration for a specific provider/gateway.

| Field    | Type                       | Description                                                                 |
|----------|----------------------------|-----------------------------------------------------------------------------|
| `Class`  | `Gateway`                  | A pointer to the gateway implementation struct (e.g. `&gateways.TermiiGateway{}`). |
| `URL`    | `string`                   | Optional: override the gateway's default API endpoint.                      |
| `Auth`   | `map[string]string`        | Authentication credentials (e.g. `{"api_key": "xxx"}`).                    |
| `Sender` | `string`                   | Optional: a gateway-specific sender ID override.                            |
| `Extras` | `map[string]interface{}`   | Optional: any extra config passed to the gateway.                           |

---

### `Place` (Geo-Routing)

Maps a short region name to a gateway and its country phone code.

| Field       | Type     | Description                                                              |
|-------------|----------|--------------------------------------------------------------------------|
| `Gateway`   | `string` | The gateway to use for this region.                                       |
| `PhoneCode` | `string` | Country dial code to prepend (e.g. `"234"` for Nigeria). Leading `+` and `0` are automatically stripped from the number before prepending. |

---

## Sending Messages

### Quick Send

The simplest way to send a message. Uses your `DefaultRoute`, `DefaultSender`, and `DefaultGateway`.

```go
result, err := client.Quick("2348012345678", "Your OTP is 1234")
```

To send without blocking (async), use `QuickAsync`:

```go
err := client.QuickAsync("2348012345678", "Your OTP is 1234")
```

> `QuickAsync` requires `Queue.Enabled = true` in the config.

---

### Fluent Builder API

For more control, use the builder API. Every method is chainable.

**Starting a message with a route:**

```go
client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("2348012345678").
        Content("Your order has shipped!").
        Line("Track it: https://example.com/track")
}).Go()
```

**Starting without a pre-set route:**

```go
client.Message().
    Route("whatsapp").
    To("2348012345678").
    Content("Hello from WhatsApp!").
    Go()
```

**Overriding the gateway per message:**

```go
client.Gateway("termii").
    To("2348012345678").
    Content("Using Termii directly").
    Go()
```

#### Available Builder Methods

| Method              | Description                                                                 |
|---------------------|-----------------------------------------------------------------------------|
| `.To("...")`        | Set one or more recipients. Separate multiple numbers with a comma: `"2348012345678, 2348098765432"`. |
| `.Content("...")`   | Set the full message body.                                                  |
| `.Line("...")`      | Append a new line to the message body. Automatically adds a `\n` separator. |
| `.Sender("...")`    | Override the sender ID for this message only.                               |
| `.Route("...")`     | Set the channel route (e.g. `"sms"`, `"whatsapp"`).                        |
| `.Gateway("...")`   | Override the gateway for this message only.                                 |

**Dispatching:**

| Method       | Description                                                           |
|--------------|-----------------------------------------------------------------------|
| `.Go()`      | Send the message **synchronously**. Returns `(interface{}, error)`.  |
| `.GoAsync()` | Push the message to the **background queue**. Returns `error`. Requires `Queue.Enabled = true`. |

---

### Async Sending

When `Queue.Enabled = true`, you can fire-and-forget messages using `.GoAsync()`. The message is pushed to the queue and a background worker will process it without blocking your main goroutine.

```go
err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.To("2348012345678").Content("Processing in the background!")
}).GoAsync()

if err != nil {
    // This only errors if the queue itself is not enabled — not from sending failure.
    log.Println("Failed to queue:", err)
}
```

Sending failures that happen inside the worker are automatically logged to stdout:
```
[Swissecho Async Error] Failed to send message: <error details>
```

---

## Routing & Channels

Routes map a channel name (like `"sms"` or `"whatsapp"`) to a set of gateways. You define them in your config and reference them by name when sending.

```go
Routes: map[string]swissecho.RouteConfig{
    "sms": {
        DefaultGateway: "termii",
        Gateways: map[string]swissecho.GatewayConfig{
            "termii": { Class: &gateways.TermiiGateway{}, ... },
        },
    },
    "whatsapp": {
        DefaultGateway: "termii",
        Gateways: map[string]swissecho.GatewayConfig{
            "termii": { Class: &gateways.TermiiGateway{}, ... },
        },
    },
},
```

You can add as many routes and gateways as you need.

---

## Geo-Routing with Places

Places let you automatically route messages to a specific gateway and apply a country code based on a short region key. This is useful when you have users across multiple countries served by different providers.

**Config:**

```go
"sms": {
    DefaultGateway: "termii",
    Gateways: map[string]swissecho.GatewayConfig{
        "termii": { ... },
        "twilio": { ... },
    },
    Places: map[string]swissecho.Place{
        "nga": {Gateway: "termii", PhoneCode: "234"},
        "usa": {Gateway: "twilio", PhoneCode: "1"},
    },
},
```

**Usage:**

```go
client.Message().
    Route("sms").
    To("08012345678").  // local number
    Go()
```

> When a `Place` is matched, the package automatically:
> 1. Selects the gateway specified in the `Place` config.
> 2. Strips any leading `+` or `0` from the phone number.
> 3. Prepends the `PhoneCode` (e.g. `08012345678` → `2348012345678`).

*Note: The place selection via the builder API is a planned enhancement. Currently, place resolution happens automatically based on `PlaceName` on the message struct.*

---

## Mock / Development Mode

Set `Enabled: false` in your config to activate mock mode. No real API calls will be made. Instead, the package intercepts the message and handles it based on the `Fake` setting.

| `Fake` value | Behaviour                                         |
|--------------|---------------------------------------------------|
| `"log"`      | Prints the message details to stdout (default).   |
| `"mail"`     | Logs a fake email to the address in `FakeMail`.   |

**Example:**

```go
config := swissecho.Config{
    Enabled:  false,      // Mock mode ON
    Fake:     "log",      // Print to stdout
    // ... rest of config
}
```

**Output:**

```
[Swissecho Mock] Mode: log | Route: sms | To: [2348012345678] | Sender: MyApp
Content: Your OTP is 1234
```

> Mock mode works transparently — your code doesn't change between development and production. Just flip the `Enabled` flag.

---

## Built-in Queue

The queue allows messages to be processed asynchronously by a pool of background workers. Errors from workers are always logged automatically.

### Memory Queue

Uses a buffered Go channel internally. Fast, zero dependencies. Messages are lost if the process restarts.

```go
Queue: swissecho.QueueConfig{
    Enabled:      true,
    QueueChannel: "memory", // or omit — this is the default
    Workers:      5,
},
```

### Redis Queue

Uses a Redis list (`LPUSH` / `BRPOP`) to persist messages across restarts. Requires a running Redis instance.

```go
Queue: swissecho.QueueConfig{
    Enabled:      true,
    QueueChannel: "redis",
    Workers:      5,
    Redis: swissecho.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",  // leave empty if no auth
        DB:       0,
    },
},
```

> The Redis queue key is `swissecho_queue`.

---

## Building a Custom Gateway

Implement the `Gateway` interface to create your own provider:

```go
type Gateway interface {
    Boot(config GatewayConfig, msg *SwissechoMessage) error
    Send() (interface{}, error)
}
```

| Method   | Description                                                                                       |
|----------|---------------------------------------------------------------------------------------------------|
| `Boot`   | Called before sending. Use this to store the config and message for use in `Send`.                |
| `Send`   | Performs the actual API call. Return the API response or an error.                                |

**Example skeleton:**

```go
package gateways

import "github.com/tekkenking/swissecho_go"

type MyGateway struct {
    config swissecho.GatewayConfig
    msg    *swissecho.SwissechoMessage
}

func (g *MyGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
    g.config = config
    g.msg = msg
    return nil
}

func (g *MyGateway) Send() (interface{}, error) {
    apiKey := g.config.Auth["api_key"]
    // Make your HTTP request here using g.msg.Recipients, g.msg.Body, g.msg.SenderID, etc.
    return nil, nil
}
```

**Register it in config:**

```go
"sms": {
    DefaultGateway: "mygateway",
    Gateways: map[string]swissecho.GatewayConfig{
        "mygateway": {
            Class: &gateways.MyGateway{},
            Auth:  map[string]string{"api_key": "your_key"},
        },
    },
},
```

---

## Included Gateways

### Termii

Handles both SMS and WhatsApp via the [Termii API](https://termii.com).

**Package:** `github.com/tekkenking/swissecho_go/gateways`

**Config:**

```go
"termii": {
    Class: &gateways.TermiiGateway{},
    Auth:  map[string]string{"api_key": "YOUR_TERMII_API_KEY"},
    // Optional: override default API endpoint
    // URL: "https://api.ng.termii.com/api/sms/send",
},
```

**Channel detection:** The Termii gateway automatically sets the channel to `"whatsapp"` when used with the `whatsapp` route, and `"generic"` for all other routes.

---

## Error Handling

All synchronous calls return `(interface{}, error)`. Always check the error:

```go
result, err := client.Quick("2348012345678", "Hello!")
if err != nil {
    // Handle the error — e.g. gateway not configured, API failure, etc.
    log.Fatal(err)
}
```

For async calls, errors during queueing (e.g. queue not enabled) are returned immediately. Errors during actual sending inside the worker are logged automatically:

```
[Swissecho Dispatch Error] Boot failed: <error>
[Swissecho Dispatch Error] Send failed: <error>
[Swissecho Async Error] Failed to send message: <error>
```
