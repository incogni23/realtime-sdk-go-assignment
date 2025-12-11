# Realtime SDK (Go)

A lightweight Go SDK for realtime messaging over WebSocket. This was built as part of a realtime messaging assignment. The goal was to keep the design clean, modular, and practical without overengineering.

## Quick Start

```go
client, _ := chat.NewClient(chat.Config{
    APIKey:   "my-key",
    Endpoint: "wss://example.com/realtime",
})

client.OnMessage("general", func(m chat.Message) {
    fmt.Println("Received:", m.Content)
})

client.Connect()
defer client.Close()

client.SendMessage("general", "Hello, world!")
```

## Core API

- `NewClient(Config)` - Create a new client instance
- `Connect()` / `Close()` - Manage connection lifecycle
- `SendMessage(channel, content)` - Send messages to a channel
- `OnMessage(channel, handler)` - Register message handlers
- `OnEvent(handler)` - Handle connection events
- `IsConnected()` - Check connection status

## Design Decisions

**Message Format**: JSON with `type`, `channel`, and `content` fields. Keeps the protocol simple while remaining extensible.

**Authentication**: API key passed via `X-API-Key` header during WebSocket handshake. Easy to modify in the transport layer if your server uses different auth.

**Concurrency**: Client is safe for concurrent use. Handlers execute on the reader goroutine, so keep them lightweight to avoid blocking message processing.

**Buffering**: 100-message channel buffer. Messages are dropped if the buffer fills. Tune this based on your throughput needs.

**Timeouts**: 5s connection timeout (configurable), 60s read deadline (fixed for now).

## Testing

Includes a mock transport for testing without a real WebSocket server:

```go
mock := chat.NewMockTransport()
client.SetTransport(mock)
mock.SimulateMessage("test", "hello")
```

See `client_test.go` for examples.

## Example

Run the example program to see message handlers, events, and graceful shutdown:

```bash
go run cmd/main.go
```

## Notes

No automatic reconnection logic yet. Connection must be manually re-established.

Read timeout should probably refresh on activity, but works fine for now.

Mock transport doesn't simulate network failures, just basic message flow.

Buffer size not yet exposed in the config (easy to add if needed).

