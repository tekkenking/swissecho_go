package swissecho_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	swissecho "github.com/tekkenking/swissecho_go"
)

// --------------------------------------------------------------------------
// Test Helpers / Fake Gateway
// --------------------------------------------------------------------------

// fakeGateway is a controllable Gateway implementation for testing.
type fakeGateway struct {
	bootCalled bool
	bootErr    error
	sendCalled bool
	sendErr    error
	lastMsg    *swissecho.SwissechoMessage
	lastConfig swissecho.GatewayConfig
}

func (f *fakeGateway) Boot(config swissecho.GatewayConfig, msg *swissecho.SwissechoMessage) error {
	f.bootCalled = true
	f.lastConfig = config
	f.lastMsg = msg
	return f.bootErr
}

func (f *fakeGateway) Send() (interface{}, error) {
	f.sendCalled = true
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return "ok", nil
}

// buildConfig creates a minimal Config with a single SMS route using the given gateway.
func buildConfig(gw swissecho.Gateway) swissecho.Config {
	return swissecho.Config{
		Enabled:       true,
		DefaultRoute:  "sms",
		DefaultSender: "TestSender",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				DefaultGateway: "fake",
				Gateways: map[string]swissecho.GatewayConfig{
					"fake": {Class: gw},
				},
			},
		},
	}
}

// --------------------------------------------------------------------------
// builder.go tests
// --------------------------------------------------------------------------

func TestMessage_To_SingleRecipient(t *testing.T) {
	msg := swissecho.NewMessage().To("2348012345678")
	if len(msg.Recipients) != 1 || msg.Recipients[0] != "2348012345678" {
		t.Fatalf("expected 1 recipient, got %v", msg.Recipients)
	}
}

func TestMessage_To_MultipleRecipients(t *testing.T) {
	msg := swissecho.NewMessage().To("2348012345678, 2348098765432")
	if len(msg.Recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %v", msg.Recipients)
	}
}

func TestMessage_To_TrimsSpaces(t *testing.T) {
	msg := swissecho.NewMessage().To("  2348012345678  ,  2348098765432  ")
	for _, r := range msg.Recipients {
		if strings.TrimSpace(r) != r {
			t.Errorf("recipient has leading/trailing spaces: %q", r)
		}
	}
}

func TestMessage_To_IgnoresEmptyParts(t *testing.T) {
	msg := swissecho.NewMessage().To(",2348012345678,")
	if len(msg.Recipients) != 1 {
		t.Fatalf("expected 1 recipient after filtering empty parts, got %v", msg.Recipients)
	}
}

func TestMessage_Content(t *testing.T) {
	msg := swissecho.NewMessage().Content("Hello!")
	if msg.Body != "Hello!" {
		t.Errorf("expected body 'Hello!', got %q", msg.Body)
	}
}

func TestMessage_Line_AppendsWithNewline(t *testing.T) {
	msg := swissecho.NewMessage().Content("Line 1").Line("Line 2")
	expected := "Line 1\nLine 2"
	if msg.Body != expected {
		t.Errorf("expected %q, got %q", expected, msg.Body)
	}
}

func TestMessage_Line_OnEmptyBody(t *testing.T) {
	msg := swissecho.NewMessage().Line("Only line")
	if msg.Body != "Only line" {
		t.Errorf("expected 'Only line', got %q", msg.Body)
	}
}

func TestMessage_Sender(t *testing.T) {
	msg := swissecho.NewMessage().Sender("MySender")
	if msg.SenderID != "MySender" {
		t.Errorf("expected sender 'MySender', got %q", msg.SenderID)
	}
}

func TestMessage_Route(t *testing.T) {
	msg := swissecho.NewMessage().Route("whatsapp")
	if msg.RouteName != "whatsapp" {
		t.Errorf("expected route 'whatsapp', got %q", msg.RouteName)
	}
}

func TestMessage_Gateway(t *testing.T) {
	msg := swissecho.NewMessage().Gateway("termii")
	if msg.GatewayName != "termii" {
		t.Errorf("expected gateway 'termii', got %q", msg.GatewayName)
	}
}

// --------------------------------------------------------------------------
// swissecho.go dispatch tests
// --------------------------------------------------------------------------

func TestQuick_CallsBootAndSend(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))

	_, err := client.Quick("2348012345678", "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gw.bootCalled {
		t.Error("expected Boot to be called")
	}
	if !gw.sendCalled {
		t.Error("expected Send to be called")
	}
}

func TestQuick_SetsDefaultSender(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))
	client.Quick("2348012345678", "Hello!")

	if gw.lastMsg.SenderID != "TestSender" {
		t.Errorf("expected sender 'TestSender', got %q", gw.lastMsg.SenderID)
	}
}

func TestQuick_MessageSenderOverridesDefault(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))
	client.Message().To("2348012345678").Content("Hello!").Sender("CustomSender").Go()

	if gw.lastMsg.SenderID != "CustomSender" {
		t.Errorf("expected sender 'CustomSender', got %q", gw.lastMsg.SenderID)
	}
}

func TestDispatch_BootError_IsReturnedAndLogged(t *testing.T) {
	gw := &fakeGateway{bootErr: fmt.Errorf("boot failed")}
	client := swissecho.New(buildConfig(gw))

	_, err := client.Quick("2348012345678", "Hello!")
	if err == nil || !strings.Contains(err.Error(), "boot failed") {
		t.Errorf("expected boot error to be returned, got %v", err)
	}
}

func TestDispatch_SendError_IsReturnedAndLogged(t *testing.T) {
	gw := &fakeGateway{sendErr: fmt.Errorf("send failed")}
	client := swissecho.New(buildConfig(gw))

	_, err := client.Quick("2348012345678", "Hello!")
	if err == nil || !strings.Contains(err.Error(), "send failed") {
		t.Errorf("expected send error to be returned, got %v", err)
	}
}

func TestDispatch_UnknownRoute_ReturnsError(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))

	_, err := client.Route("nonexistent", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("Test")
	}).Go()
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' error, got %v", err)
	}
}

func TestDispatch_MockMode_WhenDisabled(t *testing.T) {
	// When Enabled=false, the package must route to MockGateway and NOT call the real gateway.
	gw := &fakeGateway{}
	config := buildConfig(gw)
	config.Enabled = false
	config.Fake = "log"

	client := swissecho.New(config)
	_, err := client.Quick("2348012345678", "Test mock")
	if err != nil {
		t.Fatalf("unexpected error in mock mode: %v", err)
	}
	if gw.bootCalled || gw.sendCalled {
		t.Error("real gateway should NOT be called when Enabled=false")
	}
}

func TestDispatch_MockMode_NoRoutes(t *testing.T) {
	// When Enabled=false and Routes is nil/empty, it must not panic.
	config := swissecho.Config{
		Enabled: false,
		Fake:    "log",
	}
	client := swissecho.New(config)
	_, err := client.Quick("2348012345678", "Safe in mock mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --------------------------------------------------------------------------
// Phone number formatting tests
// --------------------------------------------------------------------------

func TestDispatch_PhoneFormatting_StripsLeadingPlus(t *testing.T) {
	gw := &fakeGateway{}
	config := swissecho.Config{
		Enabled:      true,
		DefaultRoute: "sms",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				DefaultGateway: "fake",
				Gateways:       map[string]swissecho.GatewayConfig{"fake": {Class: gw}},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "fake", PhoneCode: "234"},
				},
			},
		},
	}
	client := swissecho.New(config)
	msg := swissecho.NewMessage().To("+2348012345678").Content("Hello")
	msg.PlaceName = "nga"
	// Use Message() to send with a pre-built message
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("+2348012345678").Content("Hello")
	}).Go()

	// The number should not have a leading +
	for _, r := range gw.lastMsg.Recipients {
		if strings.HasPrefix(r, "+") {
			t.Errorf("recipient still has leading '+': %s", r)
		}
	}
}

func TestDispatch_PhoneFormatting_StripsAllLeadingZeros(t *testing.T) {
	gw := &fakeGateway{}
	config := swissecho.Config{
		Enabled:      true,
		DefaultRoute: "sms",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				DefaultGateway: "fake",
				Gateways:       map[string]swissecho.GatewayConfig{"fake": {Class: gw}},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "fake", PhoneCode: "234"},
				},
			},
		},
	}
	client := swissecho.New(config)
	// Simulate a Place match by calling with PlaceName on an inner message trick
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		m.PlaceName = "nga"
		return m.To("08012345678").Content("Hello")
	}).Go()

	if len(gw.lastMsg.Recipients) == 0 {
		t.Fatal("no recipients found")
	}
	// After stripping "0" and prepending "234", should be 2348012345678
	expected := "2348012345678"
	if gw.lastMsg.Recipients[0] != expected {
		t.Errorf("expected %q, got %q", expected, gw.lastMsg.Recipients[0])
	}
}

// --------------------------------------------------------------------------
// Queue tests
// --------------------------------------------------------------------------

func TestMemoryQueue_GoAsync_RequiresQueueEnabled(t *testing.T) {
	gw := &fakeGateway{}
	config := buildConfig(gw)
	config.Queue.Enabled = false
	client := swissecho.New(config)

	err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("async test")
	}).GoAsync()

	if err == nil {
		t.Error("expected an error when queue is not enabled")
	}
}

func TestMemoryQueue_GoAsync_Dispatches(t *testing.T) {
	gw := &fakeGateway{}
	config := buildConfig(gw)
	config.Queue = swissecho.QueueConfig{
		Enabled:      true,
		QueueChannel: "memory",
		Workers:      1,
	}
	client := swissecho.New(config)

	err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("async test")
	}).GoAsync()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Give the worker goroutine a moment to process
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gw.sendCalled {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !gw.sendCalled {
		t.Error("expected async worker to call Send within 2 seconds")
	}
}

func TestMemoryQueue_FullBuffer_ReturnsError(t *testing.T) {
	// Create a queue with a tiny buffer of 1 so we can easily fill it
	gw := &fakeGateway{
		// Make Send block so the channel stays full
		sendErr: fmt.Errorf("intentional slow send"),
	}
	config := buildConfig(gw)
	config.Queue = swissecho.QueueConfig{
		Enabled:      true,
		QueueChannel: "memory",
		Workers:      0, // No workers — nothing drains the channel
	}

	// We need to bypass the QueueConfig.Workers=0 default (which becomes 5)
	// so test the queue Push directly
	q := swissecho.NewMemoryQueueWithSize(1)
	msg := swissecho.NewMessage().To("1234").Content("test")

	_ = q.Push(msg) // fills the buffer

	err := q.Push(msg) // should return error, not block
	if err == nil {
		t.Error("expected error when pushing to a full queue")
	}
}

func TestQuickAsync_RequiresQueueEnabled(t *testing.T) {
	gw := &fakeGateway{}
	config := buildConfig(gw)
	config.Queue.Enabled = false
	client := swissecho.New(config)

	err := client.QuickAsync("2348012345678", "test")
	if err == nil {
		t.Error("expected error when queue is not enabled")
	}
}
