package swissecho

import "time"

// SendResult is the structured response returned after every message dispatch —
// whether it succeeds or fails. It mirrors the PHP AfterSend event payload.
type SendResult struct {
	Status          bool        // true if the send succeeded
	PartnerResponse interface{} // raw response from the gateway API
	From            string      // sender ID used
	To              []string    // recipient list (formatted)
	Body            string      // message body
	Route           string      // channel route (e.g. "sms", "whatsapp")
	Gateway         string      // gateway name used
	Identifier      interface{} // optional identifier set on the message
	Timestamp       time.Time   // time the send was attempted
	Err             error       // non-nil if the send failed
}

// AfterSendFunc is the callback signature for the post-send hook.
// It is called after every dispatch attempt, successful or not.
type AfterSendFunc func(result SendResult)
