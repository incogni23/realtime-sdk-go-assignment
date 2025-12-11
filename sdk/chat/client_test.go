package chat

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:   "test-key",
				Endpoint: "wss://example.com/realtime",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Endpoint: "wss://example.com/realtime",
			},
			wantErr: true,
		},
		{
			name: "missing endpoint",
			config: Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Use mock transport for testing
	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Errorf("Connect() error = %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}
}

func TestClient_Connect_AlreadyConnected(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.Connect()
	if err != ErrAlreadyConnected {
		t.Errorf("Connect() error = %v, want %v", err, ErrAlreadyConnected)
	}
}

func TestClient_SendMessage(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.SendMessage("general", "Hello, world!")
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	messages := mockTransport.GetSentMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].channel != "general" || messages[0].message != "Hello, world!" {
		t.Errorf("Message mismatch: got %+v", messages[0])
	}
}

func TestClient_SendMessage_NotConnected(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.SendMessage("general", "Hello, world!")
	if err != ErrNotConnected {
		t.Errorf("SendMessage() error = %v, want %v", err, ErrNotConnected)
	}
}

func TestClient_SendMessage_EmptyChannel(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.SendMessage("", "Hello, world!")
	if err != ErrChannelEmpty {
		t.Errorf("SendMessage() error = %v, want %v", err, ErrChannelEmpty)
	}
}

func TestClient_SendMessage_EmptyMessage(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.SendMessage("general", "")
	if err != ErrMessageEmpty {
		t.Errorf("SendMessage() error = %v, want %v", err, ErrMessageEmpty)
	}
}

func TestClient_OnMessage(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var receivedMessage Message
	var wg sync.WaitGroup
	wg.Add(1)

	client.OnMessage("general", func(m Message) {
		receivedMessage = m
		wg.Done()
	})

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	mockTransport.SimulateMessage("general", "Test message")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("OnMessage handler was not called")
	}

	if receivedMessage.Channel != "general" || receivedMessage.Content != "Test message" {
		t.Errorf("Handler received wrong message: got %+v", receivedMessage)
	}
}

func TestClient_OnMessage_MultipleHandlers(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var callCount int
	var mu sync.Mutex

	client.OnMessage("general", func(m Message) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	client.OnMessage("general", func(m Message) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	mockTransport.SimulateMessage("general", "Test")

	// Give handlers time to execute
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 2 {
		t.Errorf("Expected 2 handler calls, got %d", count)
	}
}

func TestClient_OnEvent(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var receivedEvents []Event
	var mu sync.Mutex

	client.OnEvent(func(e Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, e)
		mu.Unlock()
	})

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Give time for connect event
	time.Sleep(50 * time.Millisecond)

	// Simulate a message
	mockTransport.SimulateMessage("general", "Test message")

	// Give time for message event
	time.Sleep(50 * time.Millisecond)

	err = client.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Give time for disconnect event
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	events := receivedEvents
	mu.Unlock()

	// Should have at least connect, message, and disconnect events
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	// Check first event is connect
	if events[0].Type != EventConnect {
		t.Errorf("First event should be EventConnect, got %v", events[0].Type)
	}

	// Check for message event
	foundMessage := false
	for _, e := range events {
		if e.Type == EventMessage && e.Channel == "general" {
			foundMessage = true
			break
		}
	}
	if !foundMessage {
		t.Error("Message event not found")
	}
}

func TestClient_Close_WaitsForGoroutine(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Send some messages to keep the reader busy
	for i := 0; i < 5; i++ {
		mockTransport.SimulateMessage("general", "test")
	}

	// Close should wait for goroutine
	start := time.Now()
	err = client.Close()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Close should complete quickly (goroutine should finish)
	if duration > 500*time.Millisecond {
		t.Errorf("Close() took too long: %v", duration)
	}

	if client.IsConnected() {
		t.Error("Client should not be connected after Close()")
	}
}

func TestClient_SendMessage_SerializesCorrectly(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	mockTransport := NewMockTransport()
	client.SetTransport(mockTransport)

	err = client.Connect()
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err = client.SendMessage("general", "hello world")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	messages := mockTransport.GetSentMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Verify message was sent with correct channel and content
	if messages[0].channel != "general" {
		t.Errorf("Expected channel 'general', got '%s'", messages[0].channel)
	}
	if messages[0].message != "hello world" {
		t.Errorf("Expected message 'hello world', got '%s'", messages[0].message)
	}
}

func TestClient_Close_NotConnected(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestMockTransport(t *testing.T) {
	mock := NewMockTransport()

	err := mock.Connect(context.Background())
	if err != nil {
		t.Errorf("Connect() error = %v", err)
	}

	if !mock.IsConnected() {
		t.Error("Mock should be connected")
	}

	msgChan, err := mock.StartListening()
	if err != nil {
		t.Errorf("StartListening() error = %v", err)
	}

	err = mock.SendMessage(context.Background(), "test", "message")
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	messages := mock.GetSentMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	mock.SimulateMessage("test", "simulated")
	select {
	case msg := <-msgChan:
		if msg.Channel != "test" || msg.Content != "simulated" {
			t.Errorf("Received wrong message: %+v", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive simulated message")
	}

	err = mock.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}

	if mock.IsConnected() {
		t.Error("Mock should not be connected")
	}
}

func TestConfig_DefaultTimeout(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:   "test-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Check that default timeout is set
	if client.config.ConnectTimeout != defaultConnectTimeout {
		t.Errorf("Expected default timeout %v, got %v", defaultConnectTimeout, client.config.ConnectTimeout)
	}
}
