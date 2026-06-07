package swissecho

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Swissecho is the main client for interacting with the messaging service.
type Swissecho struct {
	Config    Config
	queue     DispatchQueue
	afterSend AfterSendFunc
}

// New creates a new Swissecho instance with the provided configuration.
func New(config Config) *Swissecho {
	s := &Swissecho{Config: config}

	if config.Queue.Enabled {
		if config.Queue.QueueChannel == "redis" {
			s.queue = NewRedisQueue(config.Queue.Redis)
		} else {
			s.queue = NewMemoryQueue()
		}
		s.queue.StartWorkers(config.Queue.Workers, s.dispatch)
	}

	return s
}

// OnAfterSend registers a callback that is invoked after every dispatch attempt,
// whether it succeeds or fails. Use this for logging, auditing, or alerting.
// Equivalent to listening to the PHP AfterSend event.
//
// Example:
//
//	client.OnAfterSend(func(r swissecho.SendResult) {
//	    log.Printf("Send status=%v gateway=%s to=%v", r.Status, r.Gateway, r.To)
//	})
func (s *Swissecho) OnAfterSend(fn AfterSendFunc) *Swissecho {
	s.afterSend = fn
	return s
}

// Quick sends a simple message using default settings synchronously.
func (s *Swissecho) Quick(to, content string) (SendResult, error) {
	msg := NewMessage().To(to).Content(content)
	return s.dispatch(msg)
}

// QuickAsync sends a simple message asynchronously via the configured queue.
func (s *Swissecho) QuickAsync(to, content string) error {
	msg := NewMessage().To(to).Content(content)
	if !s.Config.Queue.Enabled {
		return fmt.Errorf("queue is not enabled in config")
	}
	return s.queue.Push(msg)
}

// Gateway allows you to quickly override the default gateway.
func (s *Swissecho) Gateway(gateway string) *SwissechoRunner {
	msg := NewMessage().Gateway(gateway)
	return &SwissechoRunner{sw: s, msg: msg}
}

// Route configures a message using a specific route and a builder callback.
func (s *Swissecho) Route(routeName string, callback func(msg *SwissechoMessage) *SwissechoMessage) *SwissechoRunner {
	msg := NewMessage().Route(routeName)
	if callback != nil {
		msg = callback(msg)
	}
	return &SwissechoRunner{sw: s, msg: msg}
}

// Message starts a fluent chain without specifying a route initially.
func (s *Swissecho) Message() *SwissechoRunner {
	return &SwissechoRunner{sw: s, msg: NewMessage()}
}

// SwissechoRunner holds the state for a fluent dispatch chain.
type SwissechoRunner struct {
	sw  *Swissecho
	msg *SwissechoMessage
}

func (r *SwissechoRunner) To(to string) *SwissechoRunner {
	r.msg.To(to)
	return r
}

func (r *SwissechoRunner) Content(content string) *SwissechoRunner {
	r.msg.Content(content)
	return r
}

func (r *SwissechoRunner) Gateway(gateway string) *SwissechoRunner {
	r.msg.Gateway(gateway)
	return r
}

func (r *SwissechoRunner) Sender(sender string) *SwissechoRunner {
	r.msg.Sender(sender)
	return r
}

func (r *SwissechoRunner) Line(line string) *SwissechoRunner {
	r.msg.Line(line)
	return r
}

func (r *SwissechoRunner) Route(route string) *SwissechoRunner {
	r.msg.Route(route)
	return r
}

// Place sets the geo-routing place key for this message (e.g. "nga", "aus").
func (r *SwissechoRunner) Place(place string) *SwissechoRunner {
	r.msg.Place(place)
	return r
}

// Identifier tags this message with an arbitrary reference (e.g. a user ID).
// The identifier is included in the AfterSend callback for post-send correlation.
func (r *SwissechoRunner) Identifier(id interface{}) *SwissechoRunner {
	r.msg.IdentifierVal = id
	return r
}

// Go sends the message synchronously and waits for the response.
func (r *SwissechoRunner) Go() (SendResult, error) {
	return r.sw.dispatch(r.msg)
}

// GoAsync pushes the message to the background queue to be processed asynchronously.
func (r *SwissechoRunner) GoAsync() error {
	if !r.sw.Config.Queue.Enabled {
		return fmt.Errorf("queue is not enabled in config")
	}
	return r.sw.queue.Push(r.msg)
}

// dispatch handles the core routing and delegation to gateways.
func (s *Swissecho) dispatch(msg *SwissechoMessage) (SendResult, error) {
	// 1. Resolve Route
	routeName := msg.RouteName
	if routeName == "" {
		routeName = s.Config.DefaultRoute
	}
	if routeName == "" {
		routeName = "sms" // fallback default
	}

	routeConfig, exists := s.Config.Routes[routeName]
	if !exists {
		// Route must exist unless we are globally disabled and routing to mock anyway
		if s.Config.Enabled {
			err := fmt.Errorf("route '%s' is not configured", routeName)
			return s.failResult(msg, routeName, "", err), err
		}
	}

	// 2. Resolve Gateway
	gatewayName := msg.GatewayName
	if gatewayName == "" {
		// Geo-routing: only attempt if the route actually exists
		if exists {
			placeName := msg.PlaceName

			// 2a. Explicit place on message
			if place, ok := routeConfig.Places[placeName]; ok && placeName != "" {
				gatewayName = place.Gateway
				msg.PhoneCode = place.PhoneCode
			} else if placeName == "" && len(routeConfig.Places) > 0 {
				// 2b. Default place fallback — use the first configured place (matches PHP behaviour)
				for name, place := range routeConfig.Places {
					msg.PlaceName = name
					gatewayName = place.Gateway
					msg.PhoneCode = place.PhoneCode
					break
				}
			} else {
				gatewayName = routeConfig.DefaultGateway
			}
		}
	}

	// Global disable override → mock mode
	if !s.Config.Enabled {
		gatewayName = "mock"
	}

	var gatewayConfig GatewayConfig
	if gatewayName != "mock" {
		var gwExists bool
		gatewayConfig, gwExists = routeConfig.Gateways[gatewayName]
		if !gwExists {
			err := fmt.Errorf("gateway '%s' is not configured for route '%s'", gatewayName, routeName)
			return s.failResult(msg, routeName, gatewayName, err), err
		}
	} else {
		// Construct mock config
		gatewayConfig = GatewayConfig{
			Class:  &MockGateway{},
			Extras: map[string]interface{}{"fake": s.Config.Fake, "fake_mail": s.Config.FakeMail},
		}
	}

	if gatewayConfig.Class == nil {
		err := fmt.Errorf("gateway '%s' does not have a valid Class configured", gatewayName)
		return s.failResult(msg, routeName, gatewayName, err), err
	}

	// 3. Sender Logic
	if msg.SenderID == "" {
		msg.SenderID = gatewayConfig.Sender
		if msg.SenderID == "" {
			msg.SenderID = s.Config.DefaultSender
		}
	}

	// 4. Format numbers
	var formattedNumbers []string
	for _, num := range msg.Recipients {
		// Strip leading + sign
		num = strings.TrimPrefix(num, "+")
		// Strip all leading zeros (e.g. 0081... → 81...)
		num = strings.TrimLeft(num, "0")
		if msg.PhoneCode != "" && !strings.HasPrefix(num, msg.PhoneCode) {
			num = msg.PhoneCode + num
		}
		formattedNumbers = append(formattedNumbers, num)
	}
	msg.Recipients = formattedNumbers

	// 5. Execute
	gw := gatewayConfig.Class
	if err := gw.Boot(gatewayConfig, msg); err != nil {
		log.Printf("[Swissecho Dispatch Error] Boot failed: %v\n", err)
		return s.failResult(msg, routeName, gatewayName, err), err
	}

	apiResp, err := gw.Send()
	if err != nil {
		log.Printf("[Swissecho Dispatch Error] Send failed: %v\n", err)
		return s.failResult(msg, routeName, gatewayName, err), err
	}

	// 6. Build structured result and fire AfterSend hook
	result := SendResult{
		Status:          true,
		PartnerResponse: apiResp,
		From:            msg.SenderID,
		To:              msg.Recipients,
		Body:            msg.Body,
		Route:           routeName,
		Gateway:         gatewayName,
		Identifier:      msg.IdentifierVal,
		Timestamp:       time.Now(),
	}
	s.fireAfterSend(result)
	return result, nil
}

// failResult creates a failed SendResult and fires the AfterSend hook.
func (s *Swissecho) failResult(msg *SwissechoMessage, route, gateway string, err error) SendResult {
	result := SendResult{
		Status:     false,
		From:       msg.SenderID,
		To:         msg.Recipients,
		Body:       msg.Body,
		Route:      route,
		Gateway:    gateway,
		Identifier: msg.IdentifierVal,
		Timestamp:  time.Now(),
		Err:        err,
	}
	s.fireAfterSend(result)
	return result
}

// fireAfterSend invokes the registered AfterSend callback if one is set.
func (s *Swissecho) fireAfterSend(result SendResult) {
	if s.afterSend != nil {
		s.afterSend(result)
	}
}
