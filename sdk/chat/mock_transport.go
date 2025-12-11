package chat

import (
	"context"
	"sync"
	"time"
)

// MockTransport is a mock implementation of Transport for testing
// It can be used in examples and tests without a real server
type MockTransport struct {
	connected     bool
	mu            sync.RWMutex
	messages      []mockMessage
	msgChan       chan *Message
	shouldFail    bool
	failOnSend    bool
	failOnConnect bool
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

type mockMessage struct {
	channel string
	message string
}

// NewMockTransport creates a new mock transport for testing
// Returns the concrete type so methods like SimulateMessage can be called
func NewMockTransport() *MockTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockTransport{
		messages: make([]mockMessage, 0),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// SetShouldFail makes the mock transport fail on operations
func (m *MockTransport) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// SetFailOnSend makes the mock transport fail on SendMessage
func (m *MockTransport) SetFailOnSend(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failOnSend = fail
}

// SetFailOnConnect makes the mock transport fail on Connect
func (m *MockTransport) SetFailOnConnect(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failOnConnect = fail
}

// SimulateMessage simulates receiving a message (for testing/examples)
func (m *MockTransport) SimulateMessage(channel, content string) {
	m.mu.RLock()
	msgChan := m.msgChan
	m.mu.RUnlock()
	if msgChan != nil {
		select {
		case msgChan <- &Message{
			Type:    "message",
			Channel: channel,
			Content: content,
		}:
		default:
			// Channel full, skip
		}
	}
}

// GetSentMessages returns all messages that were sent (for testing)
func (m *MockTransport) GetSentMessages() []mockMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]mockMessage{}, m.messages...)
}

// Connect simulates connecting
func (m *MockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failOnConnect || m.shouldFail {
		return ErrConnectionFailed
	}

	if m.connected {
		return ErrAlreadyConnected
	}

	m.connected = true
	return nil
}

// Disconnect simulates disconnecting
func (m *MockTransport) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	m.cancel()
	m.wg.Wait()
	if m.msgChan != nil {
		close(m.msgChan)
	}
	return nil
}

// SendMessage simulates sending a message
func (m *MockTransport) SendMessage(ctx context.Context, channel, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failOnSend || m.shouldFail {
		return ErrSendFailed
	}

	if !m.connected {
		return ErrNotConnected
	}

	m.messages = append(m.messages, mockMessage{
		channel: channel,
		message: message,
	})
	return nil
}

// IsConnected returns the connection status
func (m *MockTransport) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// StartListening simulates starting to listen and returns a channel
func (m *MockTransport) StartListening() (<-chan *Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, ErrNotConnected
	}

	m.msgChan = make(chan *Message, 100)
	return m.msgChan, nil
}

// StopListening simulates stopping to listen
func (m *MockTransport) StopListening() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// Wait simulates waiting (useful for testing async operations)
func (m *MockTransport) Wait(duration time.Duration) {
	time.Sleep(duration)
}
