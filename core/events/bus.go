// Package events provides a pub/sub event system for StormDB v2
package events

import (
	"context"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// EventType defines the type of event
type EventType string

const (
	// Plugin events
	EventPluginLoaded   EventType = "plugin.loaded"
	EventPluginUnloaded EventType = "plugin.unloaded"
	EventPluginStarted  EventType = "plugin.started"
	EventPluginStopped  EventType = "plugin.stopped"
	EventPluginError    EventType = "plugin.error"

	// Test events
	EventTestStarted   EventType = "test.started"
	EventTestCompleted EventType = "test.completed"
	EventTestFailed    EventType = "test.failed"
	EventTestCancelled EventType = "test.cancelled"

	// System events
	EventSystemStarted EventType = "system.started"
	EventSystemStopped EventType = "system.stopped"
	EventResourceAlert EventType = "system.resource_alert"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Context   context.Context        `json:"-"`
}

// EventHandler is a function that handles events
type EventHandler func(event Event) error

// Subscription represents an event subscription
type Subscription struct {
	ID         string
	EventType  EventType
	Handler    EventHandler
	Subscriber string
	CreatedAt  time.Time
}

// EventBus provides pub/sub functionality
type EventBus struct {
	logger        core.Logger
	subscriptions map[EventType][]*Subscription
	mu            sync.RWMutex
	nextSubID     int64
	eventBuffer   []Event
	bufferSize    int
}

// NewEventBus creates a new event bus
func NewEventBus(logger core.Logger, bufferSize int) *EventBus {
	return &EventBus{
		logger:        logger.WithFields(core.Field{Key: "component", Value: "event_bus"}),
		subscriptions: make(map[EventType][]*Subscription),
		bufferSize:    bufferSize,
		eventBuffer:   make([]Event, 0, bufferSize),
	}
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event Event) error {
	eb.mu.RLock()
	subs := eb.subscriptions[event.Type]
	eb.mu.RUnlock()

	// Add to event buffer
	eb.addToBuffer(event)

	// Log the event
	eb.logger.Debug("event published",
		core.Field{Key: "event_type", Value: string(event.Type)},
		core.Field{Key: "source", Value: event.Source},
		core.Field{Key: "subscriber_count", Value: len(subs)},
	)

	// Deliver to subscribers
	for _, sub := range subs {
		go func(s *Subscription) {
			if err := s.Handler(event); err != nil {
				eb.logger.Error("event handler failed",
					core.Field{Key: "event_type", Value: string(event.Type)},
					core.Field{Key: "subscriber", Value: s.Subscriber},
					core.Field{Key: "error", Value: err.Error()},
				)
			}
		}(sub)
	}

	return nil
}

// Subscribe subscribes to events of a specific type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler, subscriber string) (*Subscription, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.nextSubID++
	sub := &Subscription{
		ID:         generateSubscriptionID(eb.nextSubID),
		EventType:  eventType,
		Handler:    handler,
		Subscriber: subscriber,
		CreatedAt:  time.Now(),
	}

	eb.subscriptions[eventType] = append(eb.subscriptions[eventType], sub)

	eb.logger.Info("event subscription created",
		core.Field{Key: "event_type", Value: string(eventType)},
		core.Field{Key: "subscriber", Value: subscriber},
		core.Field{Key: "subscription_id", Value: sub.ID},
	)

	return sub, nil
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(subscription *Subscription) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscriptions[subscription.EventType]
	for i, sub := range subs {
		if sub.ID == subscription.ID {
			// Remove from slice
			eb.subscriptions[subscription.EventType] = append(subs[:i], subs[i+1:]...)

			eb.logger.Info("event subscription removed",
				core.Field{Key: "event_type", Value: string(subscription.EventType)},
				core.Field{Key: "subscriber", Value: subscription.Subscriber},
				core.Field{Key: "subscription_id", Value: subscription.ID},
			)
			return nil
		}
	}

	return nil
}

// GetSubscriptions returns all active subscriptions
func (eb *EventBus) GetSubscriptions() map[EventType][]*Subscription {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	result := make(map[EventType][]*Subscription)
	for eventType, subs := range eb.subscriptions {
		result[eventType] = make([]*Subscription, len(subs))
		copy(result[eventType], subs)
	}
	return result
}

// GetRecentEvents returns recent events from the buffer
func (eb *EventBus) GetRecentEvents(limit int) []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if limit <= 0 || limit > len(eb.eventBuffer) {
		limit = len(eb.eventBuffer)
	}

	result := make([]Event, limit)
	startIndex := len(eb.eventBuffer) - limit
	copy(result, eb.eventBuffer[startIndex:])
	return result
}

// addToBuffer adds an event to the circular buffer
func (eb *EventBus) addToBuffer(event Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if len(eb.eventBuffer) >= eb.bufferSize {
		// Remove oldest event
		eb.eventBuffer = eb.eventBuffer[1:]
	}
	eb.eventBuffer = append(eb.eventBuffer, event)
}

// CreateEvent creates a new event with automatic ID and timestamp
func CreateEvent(eventType EventType, source string, data map[string]interface{}) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
		Context:   context.Background(),
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID(id int64) string {
	return "sub-" + time.Now().Format("20060102150405") + "-" + randomString(4)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
