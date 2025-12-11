package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/incogni23/realtime-sdk-go-assignment/sdk/chat"
)

func main() {
	// Initialize the SDK
	client, err := chat.NewClient(chat.Config{
		APIKey:   "demo-api-key",
		Endpoint: "wss://example.com/realtime",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Use mock transport for demonstration (no real server needed)
	// In production, you would use the real WebSocket transport
	mockTransport := chat.NewMockTransport()
	client.SetTransport(mockTransport)

	// Register message handler for "demo" channel
	client.OnMessage("demo", func(m chat.Message) {
		fmt.Printf("Received message on channel '%s': %s\n", m.Channel, m.Content)
	})

	// Register event handler for all events
	client.OnEvent(func(e chat.Event) {
		switch e.Type {
		case chat.EventConnect:
			fmt.Println("✓ Connected to server")
		case chat.EventDisconnect:
			fmt.Println("✗ Disconnected from server")
		case chat.EventMessage:
			fmt.Printf("📨 Message event: channel=%s, content=%s\n", e.Channel, e.Message.Content)
		case chat.EventError:
			fmt.Printf("⚠ Error event: %v\n", e.Error)
		}
	})

	// Connect to the server
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Send a message
	if err := client.SendMessage("demo", "hi"); err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	// Simulate receiving a message (using mock transport)
	time.Sleep(100 * time.Millisecond)
	mockTransport.SimulateMessage("demo", "Hello from server!")

	// Send another message
	time.Sleep(100 * time.Millisecond)
	if err := client.SendMessage("demo", "How are you?"); err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	// Simulate another incoming message
	time.Sleep(100 * time.Millisecond)
	mockTransport.SimulateMessage("demo", "I'm doing great!")

	fmt.Println("\nClient is running. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Keep the program running
	select {
	case <-sigChan:
		fmt.Println("\nShutting down...")
		if err := client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
		fmt.Println("Goodbye!")
	}
}
