package chat

import "errors"

var (
	ErrNotConnected     = errors.New("client is not connected")
	ErrAlreadyConnected = errors.New("client is already connected")
	ErrMissingAPIKey    = errors.New("missing API key")
	ErrMissingEndpoint  = errors.New("missing endpoint")
	ErrConnectionFailed = errors.New("connection failed")
	ErrSendFailed       = errors.New("failed to send message")
	ErrChannelEmpty     = errors.New("channel name cannot be empty")
	ErrMessageEmpty     = errors.New("message cannot be empty")
)
