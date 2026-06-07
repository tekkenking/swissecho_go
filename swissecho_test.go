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

func TestMessage_Content_AppendsLikePhp(t *testing.T) {
	// In PHP, content() is an alias for line() — calling it twice appends.
	msg := swissecho.NewMessage().Content("Line 1").Content("Line 2")
	expected := "Line 1\nLine 2"
	if msg.Body != expected {
		t.Errorf("expected %q, got %q", expected, msg.Body)
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

func TestMessage_Sender_TruncatesTo11Chars(t *testing.T) {
	msg := swissecho.NewMessage().Sender("VeryLongSenderIDHere")
	if len(msg.SenderID) > 11 {
		t.Errorf("expected sender to be truncated to 11 chars, got %d: %q", len(msg.SenderID), msg.SenderID)
	}
}

func TestMessage_Sender_ShortNameUnchanged(t *testing.T) {
	msg := swissecho.NewMessage().Sender("MySender")
	if msg.SenderID != "MySender" {
		t.Errorf("expected sender 'MySender', got %q", msg.SenderID)
	}
}

func TestMessage_Sender_ExactlyEleven(t *testing.T) {
	msg := swissecho.NewMessage().Sender("12345678901")
	if msg.SenderID != "12345678901" {
		t.Errorf("expected exact 11-char sender unchanged, got %q", msg.SenderID)
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

func TestMessage_Place(t *testing.T) {
	msg := swissecho.NewMessage().Place("nga")
	if msg.PlaceName != "nga" {
		t.Errorf("expected place 'nga', got %q", msg.PlaceName)
	}
}

func TestMessage_Identifier(t *testing.T) {
	msg := swissecho.NewMessage().Identifier(42)
	if msg.IdentifierVal != 42 {
		t.Errorf("expected identifier 42, got %v", msg.IdentifierVal)
	}
}

// --------------------------------------------------------------------------
// dispatch / SendResult tests
// --------------------------------------------------------------------------

func TestQuick_ReturnsStructuredSendResult(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))
	result, err := client.Quick("2348012345678", "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Status {
		t.Error("expected Status=true on success")
	}
	if result.Route != "sms" {
		t.Errorf("expected Route='sms', got %q", result.Route)
	}
	if result.Body != "Hello!" {
		t.Errorf("expected Body='Hello!', got %q", result.Body)
	}
	if result.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

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
	client.Message().To("2348012345678").Content("Hello!").Sender("Custom").Go()
	if gw.lastMsg.SenderID != "Custom" {
		t.Errorf("expected sender 'Custom', got %q", gw.lastMsg.SenderID)
	}
}

func TestDispatch_BootError_ReturnsFailedResult(t *testing.T) {
	gw := &fakeGateway{bootErr: fmt.Errorf("boot failed")}
	client := swissecho.New(buildConfig(gw))
	result, err := client.Quick("2348012345678", "Hello!")
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Status {
		t.Error("expected Status=false on failure")
	}
	if result.Err == nil {
		t.Error("expected non-nil Err on failure")
	}
	if result.Timestamp.IsZero() {
		t.Error("expected Timestamp even on failure")
	}
}

func TestDispatch_SendError_ReturnsFailedResult(t *testing.T) {
	gw := &fakeGateway{sendErr: fmt.Errorf("send failed")}
	client := swissecho.New(buildConfig(gw))
	result, err := client.Quick("2348012345678", "Hello!")
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Status {
		t.Error("expected Status=false on send failure")
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
	config := swissecho.Config{Enabled: false, Fake: "log"}
	client := swissecho.New(config)
	_, err := client.Quick("2348012345678", "Safe in mock mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --------------------------------------------------------------------------
// AfterSend hook tests
// --------------------------------------------------------------------------

func TestAfterSend_CalledOnSuccess(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))

	var captured swissecho.SendResult
	client.OnAfterSend(func(r swissecho.SendResult) { captured = r })

	client.Quick("2348012345678", "Hello")
	if !captured.Status {
		t.Error("AfterSend: expected Status=true")
	}
	if captured.Route != "sms" {
		t.Errorf("AfterSend: expected Route='sms', got %q", captured.Route)
	}
}

func TestAfterSend_CalledOnFailure(t *testing.T) {
	gw := &fakeGateway{sendErr: fmt.Errorf("network error")}
	client := swissecho.New(buildConfig(gw))

	var captured swissecho.SendResult
	client.OnAfterSend(func(r swissecho.SendResult) { captured = r })

	client.Quick("2348012345678", "Hello")
	if captured.Status {
		t.Error("AfterSend: expected Status=false on failure")
	}
	if captured.Err == nil {
		t.Error("AfterSend: expected non-nil Err on failure")
	}
}

func TestAfterSend_IncludesIdentifier(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))

	var captured swissecho.SendResult
	client.OnAfterSend(func(r swissecho.SendResult) { captured = r })

	client.Message().To("2348012345678").Content("Hello").Identifier(99).Go()
	if captured.Identifier != 99 {
		t.Errorf("AfterSend: expected Identifier=99, got %v", captured.Identifier)
	}
}

func TestAfterSend_IncludesTimestamp(t *testing.T) {
	gw := &fakeGateway{}
	client := swissecho.New(buildConfig(gw))

	var captured swissecho.SendResult
	client.OnAfterSend(func(r swissecho.SendResult) { captured = r })

	before := time.Now()
	client.Quick("2348012345678", "Hello")
	after := time.Now()

	if captured.Timestamp.Before(before) || captured.Timestamp.After(after) {
		t.Errorf("AfterSend: Timestamp %v is outside expected range [%v, %v]", captured.Timestamp, before, after)
	}
}

// --------------------------------------------------------------------------
// Place builder + default place fallback tests
// --------------------------------------------------------------------------

func TestMessage_Place_SetOnRunner(t *testing.T) {
	gw := &fakeGateway{}
	config := swissecho.Config{
		Enabled:      true,
		DefaultRoute: "sms",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				Gateways: map[string]swissecho.GatewayConfig{"fake": {Class: gw}},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "fake", PhoneCode: "234"},
				},
			},
		},
	}
	client := swissecho.New(config)
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("08012345678").Content("Hello").Place("nga")
	}).Go()
	if len(gw.lastMsg.Recipients) == 0 {
		t.Fatal("no recipients")
	}
	if gw.lastMsg.Recipients[0] != "2348012345678" {
		t.Errorf("expected '2348012345678', got %q", gw.lastMsg.Recipients[0])
	}
}

func TestDispatch_DefaultPlaceFallback(t *testing.T) {
	// When PlaceName is empty, dispatch should fall back to the first place in config
	gw := &fakeGateway{}
	config := swissecho.Config{
		Enabled:      true,
		DefaultRoute: "sms",
		Routes: map[string]swissecho.RouteConfig{
			"sms": {
				Gateways: map[string]swissecho.GatewayConfig{"fake": {Class: gw}},
				Places: map[string]swissecho.Place{
					"nga": {Gateway: "fake", PhoneCode: "234"},
				},
			},
		},
	}
	client := swissecho.New(config)
	// No .Place() called → should auto-select "nga"
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("08012345678").Content("Hello")
	}).Go()

	if len(gw.lastMsg.Recipients) == 0 {
		t.Fatal("no recipients — default place fallback not working")
	}
	if gw.lastMsg.Recipients[0] != "2348012345678" {
		t.Errorf("expected '2348012345678' from default place, got %q", gw.lastMsg.Recipients[0])
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
				Places:         map[string]swissecho.Place{"nga": {Gateway: "fake", PhoneCode: "234"}},
			},
		},
	}
	client := swissecho.New(config)
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("+2348012345678").Content("Hello").Place("nga")
	}).Go()
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
				Places:         map[string]swissecho.Place{"nga": {Gateway: "fake", PhoneCode: "234"}},
			},
		},
	}
	client := swissecho.New(config)
	client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		m.PlaceName = "nga"
		return m.To("08012345678").Content("Hello")
	}).Go()
	if len(gw.lastMsg.Recipients) == 0 {
		t.Fatal("no recipients found")
	}
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
	config.Queue = swissecho.QueueConfig{Enabled: true, QueueChannel: "memory", Workers: 1}
	client := swissecho.New(config)
	err := client.Route("sms", func(m *swissecho.SwissechoMessage) *swissecho.SwissechoMessage {
		return m.To("2348012345678").Content("async test")
	}).GoAsync()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	q := swissecho.NewMemoryQueueWithSize(1)
	msg := swissecho.NewMessage().To("1234").Content("test")
	_ = q.Push(msg)
	err := q.Push(msg)
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
