package chat

// EventType represents the type of event
type EventType string

const (
	// EventMessage represents a message event
	EventMessage EventType = "message"

	// EventConnect represents a connection event
	EventConnect EventType = "connect"

	// EventDisconnect represents a disconnection event
	EventDisconnect EventType = "disconnect"

	// EventError represents an error event
	EventError EventType = "error"
)

// Message represents an incoming message
type Message struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Content string `json:"content"`
}

// Event represents an event in the system
type Event struct {
	Type    EventType
	Channel string
	Message *Message
	Error   error
}

// MessageHandler is a function type for handling messages
type MessageHandler func(m Message)

// EventHandler is a function type for handling events
type EventHandler func(e Event)
