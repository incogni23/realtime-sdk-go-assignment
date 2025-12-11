package chat

import (
	"context"
)

type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	SendMessage(ctx context.Context, channel, message string) error
	IsConnected() bool
	StartListening() (<-chan *Message, error)
	StopListening() error
}
