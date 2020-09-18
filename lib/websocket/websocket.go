package websocket

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	// WriteWait Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second

	// PongWait Time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second

	// PingPeriod Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = (PongWait * 9) / 10

	// MaxMessageSize Maximum message size allowed from peer.
	MaxMessageSize = 512
)
var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Pool maintains clients and monitor connections.
type Pool struct {
	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

// Client is a middleman between the websocket connection and the backend.
type Client struct {
	// Pool current pool
	CurrentPool *Pool

	// CurrentConn The websocket connection.
	CurrentConn *websocket.Conn

	// Send Buffered channel of outbound messages.
	Send chan []byte
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewPool .. this will handle all websocket connections needed by SD
func NewPool() *Pool {
	return &Pool{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

// RunPool .... let just run it always
func (h *Pool) RunPool() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// UpgradeWebSocket ....
func UpgradeWebSocket(c *gin.Context) (*websocket.Conn, error) {
	webSocketWriter := c.Writer
	webSocketRequest := c.Request
	webSocketConn, err := wsupgrader.Upgrade(webSocketWriter, webSocketRequest, nil)
	if err != nil {
		log.Errorf("Failed to establish websocket connection: %v", err.Error())
		return nil, err
	}
	return webSocketConn, nil
}
