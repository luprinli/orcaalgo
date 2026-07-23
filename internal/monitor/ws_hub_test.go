package monitor

import (
	"testing"
	"time"
)

func TestNewWSHub_StartsWithNoClients(t *testing.T) {
	hub := NewWSHub()
	if hub == nil {
		t.Fatal("hub should not be nil")
	}
	hub.mu.RLock()
	count := len(hub.clients)
	hub.mu.RUnlock()
	if count != 0 {
		t.Fatalf("expected 0 clients, got %d", count)
	}
}

func TestWSHub_RegisterClient(t *testing.T) {
	hub := NewWSHub()
	client := &WSClient{send: make(chan []byte, 256), hub: hub}
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // allow goroutine to process

	hub.mu.RLock()
	if _, ok := hub.clients[client]; !ok {
		t.Fatal("client should be registered")
	}
	hub.mu.RUnlock()
}

func TestWSHub_UnregisterClient(t *testing.T) {
	hub := NewWSHub()
	client := &WSClient{send: make(chan []byte, 256), hub: hub}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	if _, ok := hub.clients[client]; ok {
		t.Fatal("client should be unregistered")
	}
	hub.mu.RUnlock()
}

func TestWSHub_BroadcastDeliversToClients(t *testing.T) {
	hub := NewWSHub()
	client := &WSClient{send: make(chan []byte, 256), hub: hub}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"channel":"risk","data":{"halted":false}}`)
	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)

	select {
	case received := <-client.send:
		if string(received) != string(msg) {
			t.Fatalf("expected %s, got %s", string(msg), string(received))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("client did not receive broadcast within timeout")
	}
}

func TestWSHub_ClientCanSendMessages(t *testing.T) {
	hub := NewWSHub()
	client := &WSClient{send: make(chan []byte, 256), hub: hub}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Simulate sending from client's channel (writePump would pick it up)
	msg := []byte(`test-message`)
	select {
	case client.send <- msg:
		// message sent to channel successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("send channel should accept message")
	}
}
