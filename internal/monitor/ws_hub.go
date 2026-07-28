package monitor

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var allowedOrigins = func() string {
	if o := os.Getenv("ORCA_WS_ORIGINS"); o != "" {
		return o
	}
	return 	"http://localhost:5173"
}()

type WSAlert struct {
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	FiredAt     string `json:"fired_at"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		for _, allowed := range strings.Split(allowedOrigins, ",") {
			if strings.TrimSpace(allowed) == origin {
				return true
			}
		}
		log.Printf("ws: rejected origin %q", origin)
		return false
	},
}

type WSClient struct {
	conn *websocket.Conn
	send chan []byte
	hub  *WSHub
}

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
	channels   map[string][]*WSClient
	started    bool
	stopFn     chan struct{}

	validateToken func(token string) bool
	onAuthFail    func()
}

func (h *WSHub) SetAuthValidator(validate func(token string) bool) {
	h.validateToken = validate
}

func (h *WSHub) SetAuthFailCallback(fn func()) {
	h.onAuthFail = fn
}

func NewWSHub() *WSHub {
	hub := &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		channels:   make(map[string][]*WSClient),
	}
	go hub.run()
	return hub
}

func (h *WSHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = auth[7:]
		}
	}

	if h.validateToken != nil && !h.validateToken(token) {
		if h.onAuthFail != nil {
			h.onAuthFail()
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	client := &WSClient{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WSClient) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

func (h *WSHub) Broadcast(channel string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msg := map[string]interface{}{
		"channel": channel,
		"data":    data,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- payload:
	default:
		RecordWSBroadcastDropped(channel)
		log.Printf("ws broadcast buffer full, dropping message for channel %s", channel)
	}
}

func (h *WSHub) SendTo(client *WSClient, channel string, data interface{}) {
	msg := map[string]interface{}{
		"channel": channel,
		"data":    data,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	client.send <- payload
}

func (h *WSHub) Clients() []*WSClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]*WSClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	return clients
}

func (h *WSHub) PushAlert(alert WSAlert) {
	h.Broadcast("alerts", alert)
}

func (h *WSHub) StartPerformanceBroadcast(interval time.Duration, getData func() interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.started {
		return
	}
	h.started = true
	h.stopFn = make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				data := getData()
				h.Broadcast("performance", data)
			case <-h.stopFn:
				return
			}
		}
	}()
}

func (h *WSHub) StopPerformanceBroadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.started {
		return
	}
	h.started = false
	close(h.stopFn)
}
