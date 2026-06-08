# Swissecho Go

A Go port of the [swissecho](https://github.com/tekkenking/swissecho) Laravel notification package.
Send SMS, WhatsApp, and Voice OTP messages through multiple provider gateways with a clean fluent API, geo-routing, async queuing, and a structured post-send callback.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
  - [Root Config](#root-config)
  - [Routes & Gateways](#routes--gateways)
  - [Geo-Routing (Places)](#geo-routing-places)
  - [Queue Config](#queue-config)
- [Sending Messages](#sending-messages)
  - [Quick Send](#quick-send)
  - [Fluent Builder](#fluent-builder)
  - [Multi-line Messages](#multi-line-messages)
  - [Override Gateway Per Message](#override-gateway-per-message)
- [Routes](#routes)
  - [SMS](#sms)
  - [WhatsApp](#whatsapp)
  - [Voice OTP](#voice-otp)
- [Geo-Routing](#geo-routing)
- [AfterSend Hook](#aftersend-hook)
- [Identifier (Audit Correlation)](#identifier-audit-correlation)
- [Async / Background Queue](#async--background-queue)
  - [Memory Queue](#memory-queue)
  - [Redis Queue](#redis-queue)
- [Mock / Development Mode](#mock--development-mode)
- [Webhook Receiver](#webhook-receiver)
- [Available Gateways](#available-gateways)
- [Sender ID Rules](#sender-id-rules)
- [Phone Number Formatting](#phone-number-formatting)
- [Running Examples](#running-examples)
- [Running Tests](#running-tests)

---

## Installation

```bash
go get github.com/tekkenking/swissecho_go
```

**Requirements:** Go 1.22+

---

## Quick Start

```go
package main

import (
    swissecho "github.com/tekkenking/swissecho_go"
    "github.com/tekkenking/swissecho_go/gateways"
)

func main() {
    config := swissecho.Config{
        Enabled:       true, // false = mock/dev mode
        DefaultRoute:  "sms",
        DefaultSender: "MyBrand",
        Routes: map[string]swissecho.RouteConfig{
            "sms": {
                DefaultGateway: "termii",
                Gateways: map[string]swissecho.GatewayConfig{
                    "termii": {
                        Class:  &gateways.TermiiGateway{},
                        Sender: "MyBrand",
                        Auth:   map[string]string{"api_key": "YOUR_KEY"},
                    },
                },
                Places: map[string]swissecho.Place{
                    "nga": {Gateway: "termii", PhoneCode: "234"},
                },
            },
        },
    }

    client := swissecho.New(config)

    // Send immediately
    result, err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
        return m.To("08012345678").Content("Your OTP is 998877").Place("nga")
    }).Go()

    if err != nil {
        panic(err)
    }
    // result.Status, result.Gateway, result.Timestamp ...
}
```

---

## Configuration Reference

### Root Config

```go
swissecho.Config{
    Enabled:       true,         // false = mock mode (no real API calls)
    Fake:          "log",        // mock output: "log" (stdout) or "mail"
    FakeMail:      "dev@you.com",// only used when Fake="mail"
    DefaultSender: "MyBrand",   // max 11 chars (enforced automatically)
    DefaultRoute:  "sms",        // used when Route() is not called
    Routes:        map[string]swissecho.RouteConfig{...},
    Queue:         swissecho.QueueConfig{...},
}
```

| Field | Type | Description |
|---|---|---|
| `Enabled` | `bool` | `true` = live sends. `false` = mock mode |
| `Fake` | `string` | `"log"` prints to stdout; `"mail"` sends an email |
| `FakeMail` | `string` | Email address for mock `"mail"` mode |
| `DefaultSender` | `string` | Fallback sender ID (max 11 chars) |
| `DefaultRoute` | `string` | Route used when no `.Route()` is specified |
| `Routes` | `map[string]RouteConfig` | One entry per channel: `"sms"`, `"whatsapp"`, `"voice"` |
| `Queue` | `QueueConfig` | Background queue settings |

---

### Routes & Gateways

```go
"sms": swissecho.RouteConfig{
    DefaultGateway: "termii",   // used when no .Gateway() override is on the message
    Gateways: map[string]swissecho.GatewayConfig{
        "termii": {
            Class:  &gateways.TermiiGateway{},
            URL:    "https://api.ng.termii.com/api/sms/send", // optional override
            Sender: "TermiiSndr",
            Auth:   map[string]string{"api_key": "YOUR_KEY"},
            Webhook: swissecho.WebhookConfig{
                Secret: "my_delivery_receipt_secret",
            },
            Extras: map[string]interface{}{
                // Gateway-specific extra fields (e.g. wirepick's "client", "affiliate")
            },
        },
    },
    Places: map[string]swissecho.Place{
        "nga": {Gateway: "termii", PhoneCode: "234"},
        "aus": {Gateway: "smsbroadcast", PhoneCode: "61"},
    },
},
```

| Field | Description |
|---|---|
| `Class` | Gateway implementation (pointer to a gateway struct) |
| `URL` | Optional URL override (most gateways have a default) |
| `Sender` | Gateway-level sender override |
| `Auth` | Map of auth credentials (`api_key`, `username`, `password`, etc.) |
| `Webhook` | Optional inbound webhook config (see [Webhook Receiver](#webhook-receiver)) |
| `Extras` | Arbitrary key-value map for gateway-specific settings |

---

### Geo-Routing (Places)

Places allow you to automatically select the right gateway and phone country code based on a named location:

```go
Places: map[string]swissecho.Place{
    "nga": {Gateway: "termii",       PhoneCode: "234"},
    "aus": {Gateway: "smsbroadcast", PhoneCode: "61"},
    "nzl": {Gateway: "tnz",          PhoneCode: "64"},
},
```

- If `.Place("nga")` is called on the message, the `termii` gateway is selected and `234` is prepended to all numbers.
- **Default place fallback**: if no `.Place()` is set, the first configured place is used automatically.

---

### Queue Config

```go
Queue: swissecho.QueueConfig{
    Enabled:      true,
    QueueChannel: "memory",  // "memory" or "redis"
    Workers:      5,         // concurrent background workers (default 5)
    Redis: swissecho.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    },
},
```

---

## Sending Messages

### Quick Send

For simple one-off sends using default route and gateway:

```go
result, err := client.Quick("2348012345678", "Hello from Swissecho!")
```

> **Note:** `Quick` does not apply place-based phone formatting. Pass the full international number without the `+`.

---

### Fluent Builder

The primary way to build and send messages:

```go
result, err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("08012345678").           // local or international format
        Content("Hello John!").      // sets body
        Place("nga").                // geo-route → selects gateway + phone code
        Sender("Alert").             // override sender (max 11 chars)
        Gateway("termii").           // override gateway
        Identifier(userID).          // tag for AfterSend callback
        Route("sms")                 // redundant here but chainable
}).Go()
```

**Builder method reference:**

| Method | Description |
|---|---|
| `.To("num1, num2")` | Set recipients. Comma-separated. Spaces trimmed. |
| `.Content("text")` | Set message body. Calling multiple times appends lines. |
| `.Line("text")` | Append a line to the body (adds `\n` separator). |
| `.Place("nga")` | Select geo-route key. Auto-selects gateway + phone code. |
| `.Sender("ID")` | Override sender ID (truncated to 11 chars automatically). |
| `.Gateway("name")` | Override the gateway for this message only. |
| `.Route("name")` | Override the route for this message only. |
| `.Identifier(val)` | Attach any value; surfaced in `AfterSend` callback. |

---

### Multi-line Messages

`Content()` and `Line()` both append — calling either multiple times builds a multi-line body:

```go
result, err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("2348012345678").
        Place("nga").
        Content("Hi John,").
        Line("Invoice #INV-001 is due tomorrow.").
        Line("Pay here: https://app.example.com/pay")
}).Go()
// Body: "Hi John,\nInvoice #INV-001 is due tomorrow.\nPay here: https://app.example.com/pay"
```

---

### Override Gateway Per Message

```go
client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("2348012345678").
        Content("Sent via NigerianBulkSMS.").
        Gateway("nigerianbulksms") // overrides the route's DefaultGateway
}).Go()
```

---

## Routes

### SMS

```go
"sms": swissecho.RouteConfig{
    DefaultGateway: "termii",
    Gateways: map[string]swissecho.GatewayConfig{
        "termii":          {Class: &gateways.TermiiGateway{},          Auth: ...},
        "nigerianbulksms": {Class: &gateways.NigerianBulkSMSGateway{}, Auth: ...},
        "routemobile":     {Class: &gateways.RouteMobileGateway{},     Auth: ...},
        "smsbroadcast":    {Class: &gateways.SMSBroadcastGateway{},    Auth: ...},
        "montnets":        {Class: &gateways.MontNetsGateway{},        Auth: ...},
        "tnz":             {Class: &gateways.TnzGateway{},             Auth: ...},
        "wirepick":        {Class: &gateways.WirepickGateway{},        Extras: ...},
    },
    Places: map[string]swissecho.Place{
        "nga": {Gateway: "termii", PhoneCode: "234"},
    },
},
```

---

### WhatsApp

```go
"whatsapp": swissecho.RouteConfig{
    DefaultGateway: "kudisms",
    Gateways: map[string]swissecho.GatewayConfig{
        "kudisms": {
            Class: &gateways.KudismsWhatsappGateway{},
            URL:   "https://app.kudisms.net/api/whatsapp",
            Auth:  map[string]string{"api_key": "YOUR_KEY"},
        },
    },
    Places: map[string]swissecho.Place{
        "nga": {Gateway: "kudisms", PhoneCode: "234"},
    },
},
```

Send:

```go
client.Route("whatsapp", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.To("2348012345678").Content("Hello from WhatsApp!").Place("nga")
}).Go()
```

---

### Voice OTP

```go
"voice": swissecho.RouteConfig{
    DefaultGateway: "termii_voice",
    Gateways: map[string]swissecho.GatewayConfig{
        "termii_voice": {
            Class: &gateways.TermiiVoiceGateway{},
            Auth:  map[string]string{"api_key": "YOUR_KEY"},
        },
        "textngxyz_voice": {
            Class: &gateways.TextngxyzVoiceGateway{},
            URL:   "https://api.textng.xyz/voice-otp/",
            Auth:  map[string]string{"api_key": "YOUR_KEY"},
            Extras: map[string]interface{}{"repeat_times": 2},
        },
    },
    Places: map[string]swissecho.Place{
        "nga": {Gateway: "termii_voice", PhoneCode: "234"},
    },
},
```

Send:

```go
client.Route("voice", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.To("2348012345678").Content("998877").Place("nga") // Content = the OTP code
}).Go()
```

---

## Geo-Routing

Set `.Place("key")` on any message to automatically select the correct gateway and prepend the country phone code:

```go
result, err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("08012345678"). // local format — 0 stripped, "234" prepended → 2348012345678
        Content("Your code is 4455").
        Place("nga")
}).Go()
```

**Fallback behaviour:** If `.Place()` is not called and no `DefaultGateway` is set, the first entry in `Places` is used automatically. If a `DefaultGateway` is set, that is used with no phone code transformation.

---

## AfterSend Hook

Register a callback that runs after **every** dispatch attempt — successful or not. Useful for logging, alerting, database records, metrics, etc.

```go
client.OnAfterSend(func(r swissecho.SendResult) {
    if r.Status {
        log.Printf("Sent OK: route=%s gateway=%s to=%v id=%v at=%s",
            r.Route, r.Gateway, r.To, r.Identifier, r.Timestamp.Format(time.RFC3339))
    } else {
        log.Printf("Send failed: %v", r.Err)
    }
})
```

**`SendResult` fields:**

| Field | Type | Description |
|---|---|---|
| `Status` | `bool` | `true` if send succeeded |
| `PartnerResponse` | `interface{}` | Raw response from the gateway API |
| `From` | `string` | Sender ID used |
| `To` | `[]string` | Final formatted recipients |
| `Body` | `string` | Message body sent |
| `Route` | `string` | Route name (`"sms"`, `"whatsapp"`, etc.) |
| `Gateway` | `string` | Gateway name used |
| `Identifier` | `interface{}` | Value set via `.Identifier()` |
| `Timestamp` | `time.Time` | When the send was attempted |
| `Err` | `error` | Non-nil on failure |

---

## Identifier (Audit Correlation)

Tag any message with a value (user ID, order ID, etc.) so you can correlate the send result in your `AfterSend` callback:

```go
client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.
        To("2348012345678").
        Content("Your order has shipped!").
        Place("nga").
        Identifier(order.ID) // any type: int, string, struct, etc.
}).Go()

// In AfterSend:
client.OnAfterSend(func(r swissecho.SendResult) {
    if r.Identifier != nil {
        orderID := r.Identifier.(int)
        db.UpdateOrderSMSStatus(orderID, r.Status)
    }
})
```

---

## Async / Background Queue

### Memory Queue

Best for single-instance deployments. Messages are held in a buffered Go channel and processed by background goroutines.

```go
config.Queue = swissecho.QueueConfig{
    Enabled:      true,
    QueueChannel: "memory",
    Workers:      5, // number of concurrent dispatch goroutines
}

client := swissecho.New(config)

// Fire-and-forget
err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
    return m.To("2348012345678").Content("Async OTP: 4433").Place("nga")
}).GoAsync()

// Or the shorthand:
err = client.QuickAsync("2348012345678", "Async OTP: 4433")
```

> The memory queue buffer holds 1000 messages. If the buffer is full, `GoAsync()` / `Push()` returns an error immediately instead of blocking.

---

### Redis Queue

Best for multi-instance deployments. Uses `go-redis` under the hood.

```go
config.Queue = swissecho.QueueConfig{
    Enabled:      true,
    QueueChannel: "redis",
    Workers:      10,
    Redis: swissecho.RedisConfig{
        Addr:     "localhost:6379",
        Password: "your_redis_password", // leave empty if no auth
        DB:       0,
    },
}

client := swissecho.New(config)
client.QuickAsync("2348012345678", "Queued via Redis!")
```

---

## Mock / Development Mode

Set `Enabled: false` to prevent any real API calls. All messages are intercepted and handled by the mock gateway.

```go
config := swissecho.Config{
    Enabled: false,
    Fake:    "log",  // prints to stdout
    // Fake: "mail", FakeMail: "dev@yourteam.com" — sends email instead
    ...
}

client := swissecho.New(config)
client.Quick("2348012345678", "This will never leave your machine")
// Output → [Swissecho Mock] Mode: log | Route: sms | To: [...] | ...
```

No routes need to be configured for mock mode — it works with an empty `Routes` map.

---

## Webhook Receiver

Receive inbound delivery receipts or status callbacks from gateway providers.

**1. Add webhook config to the gateway:**

```go
"termii": {
    Class: &gateways.TermiiGateway{},
    Auth:  map[string]string{"api_key": "YOUR_KEY"},
    Webhook: swissecho.WebhookConfig{
        Secret: "my_secret_token", // validated in the URL
    },
},
```

**2. Implement `WebhookReceiver` on your gateway (optional custom handling):**

```go
// In your own gateway or extension:
func (g *MyGateway) HandleWebhook(r *http.Request) error {
    // parse r.Body for delivery status
    return nil
}
```

**3. Mount the handler:**

```go
import "github.com/tekkenking/swissecho_go/webhook"

handler := webhook.NewHandler(config)
http.Handle("/webhook/swissecho/", handler)
http.ListenAndServe(":8080", nil)
```

**4. Point your gateway's webhook URL to:**

```
POST /webhook/swissecho/route/sms/gateway/termii/secret/my_secret_token
```

The handler validates the secret, finds the gateway, and calls `HandleWebhook()` if it is implemented.

---

## Available Gateways

All gateways live in the `gateways` package.

### SMS

| Gateway | Struct | Auth Keys | Notes |
|---|---|---|---|
| Termii | `TermiiGateway` | `api_key` | Default URL included |
| NigerianBulkSMS | `NigerianBulkSMSGateway` | `username`, `password` | Default URL included |
| RouteMobile | `RouteMobileGateway` | `username`, `password` | URL required |
| SMSBroadcast | `SMSBroadcastGateway` | `username`, `password` | URL required |
| MontNets | `MontNetsGateway` | `username`, `password` | MD5 password hashing built in |
| TNZ | `TnzGateway` | `api_key` | Basic Auth header |
| Wirepick | `WirepickGateway` | — | Uses `Extras`: `client`, `password`, `affiliate` |

### WhatsApp

| Gateway | Struct | Auth Keys |
|---|---|---|
| Kudisms WhatsApp | `KudismsWhatsappGateway` | `api_key` |
| Termii WhatsApp | `TermiiWhatsappGateway` | `api_key` |
| Direct WhatsApp | `DirectWhatsappGateway` | `username`, `password` |

### Voice OTP

| Gateway | Struct | Auth Keys | Notes |
|---|---|---|---|
| Termii Voice | `TermiiVoiceGateway` | `api_key` | Default URL included |
| TextngXYZ Voice | `TextngxyzVoiceGateway` | `api_key` | Extras: `repeat_times` (int) |

---

## Sender ID Rules

Sender IDs (alphanumeric) are limited to **11 characters** by most networks. Swissecho enforces this automatically:

```go
// Input:   "VeryLongCompanyName"
// Stored:  "VeryLongCom"  (truncated silently to 11 chars)
msg := swissecho.NewMessage().Sender("VeryLongCompanyName")
```

Set your sender at the gateway level (`GatewayConfig.Sender`) for a per-gateway default, or at the root level (`Config.DefaultSender`) for the global fallback.

---

## Phone Number Formatting

When a `Place` is matched (either explicitly via `.Place()` or via the default-place fallback), Swissecho automatically:

1. Strips a leading `+` if present.
2. Strips all leading `0`s.
3. Prepends the place's `PhoneCode`.
4. Skips prepending if the number already starts with the phone code.

```
Input:  "+2348012345678"  →  "2348012345678"   (+ stripped, code already present)
Input:  "08012345678"     →  "2348012345678"   (0 stripped, 234 prepended)
Input:  "8012345678"      →  "2348012345678"   (234 prepended)
```

If no place is configured, numbers are sent as-is.

---

## Running Examples

```bash
git clone https://github.com/tekkenking/swissecho_go
cd swissecho_go
GOTOOLCHAIN=local CGO_ENABLED=0 go run ./examples/
```

All 12 examples run in mock mode — no API keys needed.

---

## Running Tests

```bash
GOTOOLCHAIN=local CGO_ENABLED=0 go test ./... -v
```

The test suite covers:
- Builder methods (To, Content, Line, Sender, Place, Gateway, Route, Identifier)
- Sender ID 11-char enforcement
- Content/Line append semantics
- Dispatch routing and gateway resolution
- Mock mode (with and without routes)
- Boot and Send error handling → structured `SendResult`
- `AfterSend` callback (success, failure, identifier, timestamp)
- `.Place()` geo-routing on message builder
- Default place fallback
- Phone number formatting (strip `+`, strip leading zeros, prepend code)
- Async memory queue (GoAsync, QuickAsync)
- Full queue error on buffer overflow

---

## License

MIT
