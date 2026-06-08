// Package main demonstrates every major feature of swissecho_go.
//
// Run with:
//
//	go run ./examples/
//
// All examples use Enabled:false (mock mode) so no real API calls are made.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	swissecho "github.com/tekkenking/swissecho_go"
	"github.com/tekkenking/swissecho_go/gateways"
	"github.com/tekkenking/swissecho_go/webhook"
)

// ─────────────────────────────────────────────
// 1.  Build the shared config (mock mode)
// ─────────────────────────────────────────────
func buildConfig() swissecho.Config {
	return swissecho.Config{
		// Set Enabled: true in production once you have real API keys.
		Enabled:       false,
		Fake:          "log", // "log" prints to stdout; "mail" would send an email
		DefaultRoute:  "sms",
		DefaultSender: "MyBrand", // Max 11 chars enforced automatically

		Routes: map[string]swissecho.RouteConfig{

			// ── SMS ──────────────────────────────────────────────────────
			"sms": {
				DefaultGateway: "termii",
				Gateways: map[string]swissecho.GatewayConfig{
					"termii": {
						Class:  &gateways.TermiiGateway{},
						URL:    "https://api.ng.termii.com/api/sms/send", // optional override
						Sender: "Termii",
						Auth:   map[string]string{"api_key": "YOUR_TERMII_KEY"},
						Webhook: swissecho.WebhookConfig{
							Secret: "my_webhook_secret",
						},
					},
					"nigerianbulksms": {
						Class:  &gateways.NigerianBulkSMSGateway{},
						URL:    "https://portal.nigeriabulksms.com/api/",
						Sender: "MyBrand",
						Auth: map[string]string{
							"username": "YOUR_USERNAME",
							"password": "YOUR_PASSWORD",
						},
					},
					"routemobile": {
						Class:  &gateways.RouteMobileGateway{},
						URL:    "https://your.routemobile.url/",
						Sender: "MyBrand",
						Auth: map[string]string{
							"username": "YOUR_USERNAME",
							"password": "YOUR_PASSWORD",
						},
					},
					"smsbroadcast": {
						Class:  &gateways.SMSBroadcastGateway{},
						URL:    "https://www.smsbroadcast.com.au/api-adv.php",
						Sender: "MyBrand",
						Auth: map[string]string{
							"username": "YOUR_USERNAME",
							"password": "YOUR_PASSWORD",
						},
					},
					"montnets": {
						Class: &gateways.MontNetsGateway{},
						URL:   "https://sd.montnets.com/sms",
						Auth: map[string]string{
							"username": "YOUR_USERNAME",
							"password": "YOUR_PASSWORD",
						},
					},
					"tnz": {
						Class: &gateways.TnzGateway{},
						URL:   "https://api.tnz.co.nz/api/v2.01/SENDER/sms",
						Auth:  map[string]string{"api_key": "YOUR_TNZ_KEY"},
					},
					"wirepick": {
						Class: &gateways.WirepickGateway{},
						URL:   "https://wirepick.url/send",
						Extras: map[string]interface{}{
							"client":    "YOUR_CLIENT",
							"password":  "YOUR_PASSWORD",
							"affiliate": "YOUR_AFFILIATE",
						},
					},
				},
				// Places = geo-routing.  First entry is the automatic default.
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "termii", PhoneCode: "234"},
					"aus": {Gateway: "smsbroadcast", PhoneCode: "61"},
					"nzl": {Gateway: "tnz", PhoneCode: "64"},
				},
			},

			// ── WhatsApp ─────────────────────────────────────────────────
			"whatsapp": {
				DefaultGateway: "kudisms",
				Gateways: map[string]swissecho.GatewayConfig{
					"kudisms": {
						Class: &gateways.KudismsWhatsappGateway{},
						URL:   "https://app.kudisms.net/api/whatsapp",
						Auth:  map[string]string{"api_key": "YOUR_KUDISMS_KEY"},
					},
				},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "kudisms", PhoneCode: "234"},
				},
			},

			// ── Voice OTP ─────────────────────────────────────────────────
			"voice": {
				DefaultGateway: "termii_voice",
				Gateways: map[string]swissecho.GatewayConfig{
					"termii_voice": {
						Class: &gateways.TermiiVoiceGateway{},
						Auth:  map[string]string{"api_key": "YOUR_TERMII_KEY"},
					},
					"textngxyz_voice": {
						Class: &gateways.TextngxyzVoiceGateway{},
						URL:   "https://api.textng.xyz/voice-otp/",
						Auth:  map[string]string{"api_key": "YOUR_TEXTNG_KEY"},
						Extras: map[string]interface{}{
							"repeat_times": 2,
						},
					},
				},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "termii_voice", PhoneCode: "234"},
				},
			},
		},
	}
}

func main() {
	config := buildConfig()
	client := swissecho.New(config)

	// ─── Register a post-send hook (AfterSend) ───────────────────────────────
	// This is called after EVERY dispatch attempt — success or failure.
	// Use it for logging, alerting, storing audit records, etc.
	client.OnAfterSend(func(r swissecho.SendResult) {
		if r.Status {
			fmt.Printf("[AfterSend] ✓ Sent via %s/%s to %v\n", r.Route, r.Gateway, r.To)
		} else {
			fmt.Printf("[AfterSend] ✗ Failed via %s/%s — %v\n", r.Route, r.Gateway, r.Err)
		}
		if r.Identifier != nil {
			fmt.Printf("[AfterSend]   Identifier: %v\n", r.Identifier)
		}
	})

	separator := func(title string) {
		fmt.Printf("\n%s\n%s\n", title, "────────────────────────────────────────")
	}

	// ─── Example 1: Quick send (one-liner) ──────────────────────────────────
	separator("1. Quick send")
	// For Quick, phone numbers must already be in international format (no prefix stripping needed).
	result, err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("Your OTP is 123456").Place("nga")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	} else {
		fmt.Printf("Status: %v | From: %s | To: %v\n", result.Status, result.From, result.To)
	}

	// ─── Example 2: Fluent builder with Place (geo-routing) ──────────────────
	separator("2. Fluent builder — geo-routed to Nigeria")
	_, err = client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("08012345678"). // local format; phone code prepended automatically
			Content("Welcome to our service!").
			Place("nga"). // selects termii gateway + adds 234 prefix
			Sender("Alert")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 3: Multi-line message ───────────────────────────────────────
	separator("3. Multi-line message")
	_, err = client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("2348012345678").
			Place("nga").
			Content("Hello John,").
			Line("Your invoice #INV-00100 is due in 3 days.").
			Line("Login to pay: https://app.example.com")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 4: WhatsApp route ───────────────────────────────────────────
	separator("4. WhatsApp route")
	_, err = client.Route("whatsapp", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("2348012345678").
			Content("Hello from WhatsApp!").
			Place("nga")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 5: Voice OTP ────────────────────────────────────────────────
	separator("5. Voice OTP")
	_, err = client.Route("voice", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("2348012345678").
			Content("9988"). // the OTP code to be read out
			Place("nga")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 6: Message with Identifier (AfterSend correlation) ──────────
	separator("6. Identifier for audit/AfterSend correlation")
	type User struct{ ID int; Name string }
	user := User{ID: 42, Name: "Ada"}
	_, err = client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("2348012345678").
			Content(fmt.Sprintf("Hi %s, your account is active.", user.Name)).
			Place("nga").
			Identifier(user.ID) // surfaced in AfterSend callback
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 7: Override gateway per message ─────────────────────────────
	separator("7. Per-message gateway override")
	_, err = client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.
			To("2348012345678").
			Content("Sent via NigerianBulkSMS specifically.").
			Gateway("nigerianbulksms")
	}).Go()
	if err != nil {
		log.Println("Error:", err)
	}

	// ─── Example 8: Async via memory queue ───────────────────────────────────
	separator("8. Async — memory queue")

	asyncConfig := buildConfig()
	asyncConfig.Queue = swissecho.QueueConfig{
		Enabled:      true,
		QueueChannel: "memory",
		Workers:      3,
	}
	asyncClient := swissecho.New(asyncConfig)
	asyncClient.OnAfterSend(func(r swissecho.SendResult) {
		fmt.Printf("[AsyncAfterSend] %s/%s → status=%v\n", r.Route, r.Gateway, r.Status)
	})

	err = asyncClient.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("This processes in the background!").Place("nga")
	}).GoAsync()
	if err != nil {
		log.Println("Async error:", err)
	} else {
		fmt.Println("Message queued. Worker will process it shortly...")
	}

	// Also test QuickAsync helper — pass full international number
	err = asyncClient.QuickAsync("2348099999999", "Quick async OTP: 5544")
	if err != nil {
		log.Println("QuickAsync error:", err)
	}

	// ─── Example 9: Async via Redis queue (commented out — needs Redis) ───────
	separator("9. Async — Redis queue (requires running Redis)")
	fmt.Println("(Skipped in this demo — uncomment to use)")
	/*
		redisConfig := buildConfig()
		redisConfig.Queue = swissecho.QueueConfig{
			Enabled:      true,
			QueueChannel: "redis",
			Workers:      5,
			Redis: swissecho.RedisConfig{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			},
		}
		redisClient := swissecho.New(redisConfig)
		redisClient.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
			return m.To("2348012345678").Content("Queued via Redis!")
		}).GoAsync()
	*/

	// ─── Example 10: Webhook receiver setup ──────────────────────────────────
	separator("10. Webhook receiver (HTTP server — not started in this demo)")
	fmt.Println("Mount the webhook handler in your HTTP server:")
	fmt.Println(`
    handler := webhook.NewHandler(config)
    http.Handle("/webhook/swissecho/", handler)
    // Receive delivery receipts at:
    // POST /webhook/swissecho/route/sms/gateway/termii/secret/my_webhook_secret
	`)
	_ = webhook.NewHandler(config) // proves it compiles

	// ─── Example 11: Sender ID auto-truncation ───────────────────────────────
	separator("11. Sender ID — auto-truncated to 11 chars")
	msg := swissecho.NewMessage().Sender("VeryLongCompanyNameHere")
	fmt.Printf("Input: 'VeryLongCompanyNameHere' → Stored as: '%s' (%d chars)\n",
		msg.SenderID, len(msg.SenderID))

	// ─── Example 12: Structured return value ─────────────────────────────────
	separator("12. Structured SendResult")
	res, _ := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("Checking result fields").Place("nga")
	}).Go()
	fmt.Printf("Status:    %v\n", res.Status)
	fmt.Printf("Route:     %s\n", res.Route)
	fmt.Printf("Gateway:   %s\n", res.Gateway)
	fmt.Printf("From:      %s\n", res.From)
	fmt.Printf("To:        %v\n", res.To)
	fmt.Printf("Timestamp: %s\n", res.Timestamp.Format(time.RFC3339))

	// ─── Wait for async workers to drain ─────────────────────────────────────
	fmt.Println("\n(waiting 1s for async workers to finish...)")
	time.Sleep(1 * time.Second)
	fmt.Println("\n✅ All examples completed.")

	// Prevent unused import error for net/http
	_ = http.StatusOK
}
