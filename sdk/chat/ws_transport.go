package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsTransport implements the Transport interface using WebSocket
type wsTransport struct {
	endpoint string
	apiKey   string
	conn     *websocket.Conn
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	msgChan  chan *Message
}

// NewWebSocketTransport creates a new WebSocket transport instance
func NewWebSocketTransport(endpoint, apiKey string) Transport {
	ctx, cancel := context.WithCancel(context.Background())
	return &wsTransport{
		endpoint: endpoint,
		apiKey:   apiKey,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Connect establishes a WebSocket connection
func (w *wsTransport) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return ErrAlreadyConnected
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second, // TODO: make this configurable?
	}

	headers := make(map[string][]string)
	headers["X-API-Key"] = []string{w.apiKey}

	conn, _, err := dialer.DialContext(ctx, w.endpoint, headers)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	w.conn = conn
	return nil
}

// Disconnect closes the WebSocket connection
func (w *wsTransport) Disconnect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn == nil {
		return nil
	}

	w.cancel()
	err := w.conn.Close()
	w.conn = nil
	w.wg.Wait()
	if w.msgChan != nil {
		close(w.msgChan)
	}
	return err
}

// SendMessage sends a message through the WebSocket connection
func (w *wsTransport) SendMessage(ctx context.Context, channel, message string) error {
	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	payload := map[string]interface{}{
		"type":    "message",
		"channel": channel,
		"content": message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return nil
}

// IsConnected returns whether the transport is connected
func (w *wsTransport) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.conn != nil
}

// StartListening starts listening for incoming messages and returns a channel
func (w *wsTransport) StartListening() (<-chan *Message, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn == nil {
		return nil, ErrNotConnected
	}

	w.msgChan = make(chan *Message, 100) // buffer size might need tuning
	conn := w.conn

	w.wg.Add(1)
	go w.listen(conn)

	return w.msgChan, nil
}

func (w *wsTransport) listen(conn *websocket.Conn) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return
				}
				return
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			select {
			case w.msgChan <- &msg:
			case <-w.ctx.Done():
				return
			default:
				// channel full, drop message
				// TODO: maybe add metrics/logging here?
			}
		}
	}
}

// StopListening stops listening for messages
func (w *wsTransport) StopListening() error {
	w.cancel()
	w.wg.Wait()
	return nil
}
