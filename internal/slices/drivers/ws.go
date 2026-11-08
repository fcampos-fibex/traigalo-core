package drivers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type OrderTrackingEvent struct {
	OrderID   *uuid.UUID `json:"order_id,omitempty"`
	DriverID  uuid.UUID  `json:"driver_id"`
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Timestamp time.Time  `json:"timestamp"`
}

type Hub struct {
	redis          *redis.Client
	clients        map[*Client]bool
	orderClients   map[string]map[*Client]bool
	register       chan *Client
	unregister     chan *Client
	broadcastOrder chan OrderTrackingEvent
	mu             sync.RWMutex
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	orderID string
}

func NewHub(redis *redis.Client) *Hub {
	return &Hub{
		redis:          redis,
		clients:        make(map[*Client]bool),
		orderClients:   make(map[string]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcastOrder: make(chan OrderTrackingEvent, 256),
	}
}

func (h *Hub) Run(ctx context.Context) {
	if h.redis != nil {
		go h.subscribeRedis(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				_ = client.conn.Close()
				delete(h.clients, client)
			}
			h.orderClients = make(map[string]map[*Client]bool)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.orderID != "" {
				if _, ok := h.orderClients[client.orderID]; !ok {
					h.orderClients[client.orderID] = make(map[*Client]bool)
				}
				h.orderClients[client.orderID][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.orderID != "" && h.orderClients[client.orderID] != nil {
					delete(h.orderClients[client.orderID], client)
					if len(h.orderClients[client.orderID]) == 0 {
						delete(h.orderClients, client.orderID)
					}
				}
			}
			h.mu.Unlock()

		case event := <-h.broadcastOrder:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			h.mu.RLock()
			if event.OrderID != nil {
				orderKey := event.OrderID.String()
				if targets, ok := h.orderClients[orderKey]; ok {
					for c := range targets {
						select {
						case c.send <- data:
						default:
							close(c.send)
							delete(h.clients, c)
							delete(targets, c)
						}
					}
				}
			}

			for c := range h.clients {
				if c.orderID == "" {
					select {
					case c.send <- data:
					default:
						close(c.send)
						delete(h.clients, c)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) subscribeRedis(ctx context.Context) {
	pubsub := h.redis.Subscribe(ctx, LocationUpdatesChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var event OrderTrackingEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err == nil {
				h.broadcastOrder <- event
			}
		}
	}
}

func (h *Hub) BroadcastLocation(event OrderTrackingEvent) {
	h.broadcastOrder <- event
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	orderID := r.URL.Query().Get("order-id")
	if orderID == "" {
		orderID = r.URL.Query().Get("order_id")
	}

	client := &Client{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, 64),
		orderID: orderID,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
