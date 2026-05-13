package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/ayan-de/agent-board/internal/core"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	sessID string
}

type wsHub struct {
	clients    map[*wsClient]bool
	register   chan *wsClient
	unregister chan *wsClient
	broadcast  chan []byte
	mu         sync.RWMutex
	sessID     string
}

func newHub(sessionID string) *wsHub {
	return &wsHub{
		clients:    make(map[*wsClient]bool),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		broadcast:  make(chan []byte, 256),
		sessID:     sessionID,
	}
}

func (h *wsHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("ws client connected (total: %d)", len(h.clients))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("ws client disconnected (total: %d)", len(h.clients))
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *wsHub) broadcastCompletion(comp core.RunCompletion) {
	data, err := json.Marshal(map[string]interface{}{
		"type": "run_completion",
		"data": comp,
	})
	if err != nil {
		return
	}
	h.broadcast <- data
}

type HubManager struct {
	mu    sync.RWMutex
	hubs  map[string]*wsHub
	compCh <-chan core.RunCompletion
}

func NewHubManager(compCh <-chan core.RunCompletion) *HubManager {
	hm := &HubManager{
		hubs:   make(map[string]*wsHub),
		compCh: compCh,
	}
	go hm.forwardCompletions()
	return hm
}

func (hm *HubManager) forwardCompletions() {
	for comp := range hm.compCh {
		hm.mu.RLock()
		hub, ok := hm.hubs[comp.SessionID]
		hm.mu.RUnlock()
		if ok {
			hub.broadcastCompletion(comp)
		}
	}
}

func (hm *HubManager) GetHub(sessionID string) *wsHub {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hub, ok := hm.hubs[sessionID]; ok {
		return hub
	}
	hub := newHub(sessionID)
	hm.hubs[sessionID] = hub
	go hub.Run()
	return hub
}

func (hm *HubManager) RemoveHub(sessionID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.hubs, sessionID)
}

func (h *wsHub) ServeWs(w http.ResponseWriter, r *http.Request, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	client := &wsClient{
		conn:   conn,
		send:   make(chan []byte, 256),
		sessID: sessionID,
	}
	h.register <- client

	go client.writePump()
	go client.readPump(h.unregister)
}

func (c *wsClient) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *wsClient) readPump(unregister chan<- *wsClient) {
	defer func() {
		unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
