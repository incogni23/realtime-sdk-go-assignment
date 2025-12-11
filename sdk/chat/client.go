package chat

import (
	"context"
	"sync"
)

// Client represents the main SDK client
type Client struct {
	config    Config
	transport Transport
	mu        sync.RWMutex
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Handler registry
	messageHandlers map[string][]MessageHandler // channel -> handlers
	eventHandlers   []EventHandler
	handlersMu      sync.RWMutex
	messageChan     <-chan *Message
}

// NewClient creates a new chat client with the provided configuration
func NewClient(config Config) (*Client, error) {
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = defaultConnectTimeout
	}

	if config.APIKey == "" {
		return nil, ErrMissingAPIKey
	}
	if config.Endpoint == "" {
		return nil, ErrMissingEndpoint
	}

	ctx, cancel := context.WithCancel(context.Background())
	transport := NewWebSocketTransport(config.Endpoint, config.APIKey, config.Logger)

	client := &Client{
		config:          config,
		transport:       transport,
		ctx:             ctx,
		cancel:          cancel,
		messageHandlers: make(map[string][]MessageHandler),
		eventHandlers:   make([]EventHandler, 0),
	}

	return client, nil
}

// OnMessage registers a handler for messages on a specific channel
func (c *Client) OnMessage(channel string, handler MessageHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.messageHandlers[channel] = append(c.messageHandlers[channel], handler)
}

// OnEvent registers a handler for all events
func (c *Client) OnEvent(handler EventHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Connect establishes a connection to the server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return ErrAlreadyConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConnectTimeout)
	defer cancel()

	if err := c.transport.Connect(ctx); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Error("failed to connect", "error", err)
		}
		return err
	}

	messageChan, err := c.transport.StartListening()
	if err != nil {
		c.transport.Disconnect()
		if c.config.Logger != nil {
			c.config.Logger.Error("failed to start listening", "error", err)
		}
		return err
	}

	c.messageChan = messageChan
	c.connected = true

	c.wg.Add(1)
	go c.reader()

	c.emitEvent(Event{
		Type: EventConnect,
	})

	if c.config.Logger != nil {
		c.config.Logger.Info("connected to server")
	}

	return nil
}

func (c *Client) reader() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.messageChan:
			if !ok {
				c.emitEvent(Event{
					Type: EventDisconnect,
				})
				return
			}
			if msg != nil {
				c.handleMessage(msg)
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	// copy message for handlers (they get value, not pointer)
	message := Message{
		Type:    msg.Type,
		Channel: msg.Channel,
		Content: msg.Content,
	}

	c.emitEvent(Event{
		Type:    EventMessage,
		Channel: msg.Channel,
		Message: msg,
	})

	c.handlersMu.RLock()
	handlers := c.messageHandlers[msg.Channel]
	c.handlersMu.RUnlock()

	for _, h := range handlers {
		h(message)
	}
}

func (c *Client) emitEvent(event Event) {
	c.handlersMu.RLock()
	handlers := make([]EventHandler, len(c.eventHandlers))
	copy(handlers, c.eventHandlers)
	c.handlersMu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// SendMessage sends a message to the specified channel
func (c *Client) SendMessage(channel, message string) error {
	if channel == "" {
		return ErrChannelEmpty
	}
	if message == "" {
		return ErrMessageEmpty
	}

	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected {
		return ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConnectTimeout)
	defer cancel()

	err := transport.SendMessage(ctx, channel, message)
	return err
}

// Close gracefully shuts down the client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.cancel()

	if err := c.transport.StopListening(); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Error("error stopping listener", "error", err)
		}
	}

	if err := c.transport.Disconnect(); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Error("error disconnecting", "error", err)
		}
	}

	c.connected = false
	c.wg.Wait()

	c.emitEvent(Event{
		Type: EventDisconnect,
	})

	if c.config.Logger != nil {
		c.config.Logger.Info("disconnected from server")
	}

	return nil
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.transport.IsConnected()
}

// SetTransport allows setting a custom transport for testing
func (c *Client) SetTransport(transport Transport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transport = transport
}
