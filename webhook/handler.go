package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	swissecho "github.com/tekkenking/swissecho_go"
)

// WebhookHandler is an http.Handler that receives inbound callbacks from gateway
// providers (e.g. delivery receipts, status updates).
//
// URL format: /webhook/swissecho/route/{route}/gateway/{gateway}/secret/{secret}
//
// Example registration with net/http:
//
//	handler := webhook.NewHandler(client.Config)
//	http.Handle("/webhook/swissecho/", handler)
//
// Example registration with gorilla/mux:
//
//	r.HandleFunc("/webhook/swissecho/route/{route}/gateway/{gateway}/secret/{secret}",
//	    webhook.NewHandler(config).ServeHTTP)
type WebhookHandler struct {
	config swissecho.Config
}

// NewHandler creates a new WebhookHandler using your Swissecho config.
func NewHandler(config swissecho.Config) *WebhookHandler {
	return &WebhookHandler{config: config}
}

// ServeHTTP validates the secret and forwards the webhook payload to the
// gateway's registered WebhookReceiver, if it implements one.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse path: /webhook/swissecho/route/{route}/gateway/{gateway}/secret/{secret}
	routeName, gatewayName, secret, err := parsePath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Lookup route
	routeConfig, ok := h.config.Routes[routeName]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("route '%s' not found", routeName)})
		return
	}

	// Lookup gateway
	gwConfig, ok := routeConfig.Gateways[gatewayName]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("gateway '%s' not found on route '%s'", gatewayName, routeName)})
		return
	}

	// Validate secret
	if gwConfig.Webhook.Secret == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no webhook configured for this gateway"})
		return
	}
	if secret != gwConfig.Webhook.Secret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid webhook secret"})
		return
	}

	// Check if gateway implements WebhookReceiver
	receiver, ok := gwConfig.Class.(swissecho.WebhookReceiver)
	if !ok {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "gateway does not implement webhook handling"})
		return
	}

	// Forward to gateway
	if err := receiver.HandleWebhook(r); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "webhook received"})
}

// parsePath extracts route, gateway, secret from:
// /webhook/swissecho/route/{route}/gateway/{gateway}/secret/{secret}
func parsePath(path string) (route, gateway, secret string, err error) {
	// Simple segment parser
	segments := splitPath(path)
	// Expected: ["", "webhook", "swissecho", "route", {route}, "gateway", {gateway}, "secret", {secret}]
	if len(segments) < 9 {
		return "", "", "", fmt.Errorf("invalid webhook URL format")
	}
	return segments[4], segments[6], segments[8], nil
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	return parts
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
