package chat

import "time"

type Config struct {
	Endpoint       string
	APIKey         string
	ConnectTimeout time.Duration
	Logger         Logger
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

const (
	defaultConnectTimeout = 5 * time.Second
)
