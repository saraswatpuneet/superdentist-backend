package websocket

import (
	"net/http"
	"strings"
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
	Clients map[string]*Client

	// Inbound messages from the clients.
	Broadcast chan BroadCastChannel

	// Register requests from the clients.
	Register chan *RegisterChannel

	// Unregister requests from clients.
	Unregister chan *UnRegisterChannel
}

// BroadCastChannel ...
type BroadCastChannel struct {
	ClientID string
	Message  []byte
}

// RegisterChannel ...
type RegisterChannel struct {
	ClientID  string
	WebClient *Client
}

// UnRegisterChannel ...
type UnRegisterChannel struct {
	ClientID  string
	WebClient *Client
}

// Client is a middleman between the websocket connection and the backend.
type Client struct {
	// Pool current pool
	CurrentPool *Pool

	// CurrentConn The websocket connection.
	CurrentConn *websocket.Conn

	//CurrentConnID ... the id of connection established
	CurrentConnID string

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
		Broadcast:  make(chan BroadCastChannel),
		Register:   make(chan *RegisterChannel),
		Unregister: make(chan *UnRegisterChannel),
		Clients:    make(map[string]*Client),
	}
}

// RunPool .... let just run it always
func (h *Pool) RunPool() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.ClientID] = client.WebClient
		case client := <-h.Unregister:
			if _, ok := h.Clients[client.ClientID]; ok {
				delete(h.Clients, client.ClientID)
				close(client.WebClient.Send)
			}
		case message := <-h.Broadcast:
			clientID := message.ClientID
			if client, ok := h.Clients[clientID]; ok {
				client.Send <- message.Message
			}
		}
	}
}

// UpgradeWebSocket ....
func UpgradeWebSocket(c *gin.Context) (*websocket.Conn, error) {
	webSocketWriter := c.Writer
	webSocketRequest := c.Request
	wsupgrader.CheckOrigin = func(r *http.Request) bool { return true }
	respHeader := make(http.Header, 0)
	for key, value := range c.Request.Header {
		if strings.ToLower(key)!="sec-webSocket-extensions" {
			respHeader[key] = value
		}
	}
	respnHeader := c.Request.Header
	webSocketConn, err := wsupgrader.Upgrade(webSocketWriter, webSocketRequest,respnHeader)
	if err != nil {
		log.Errorf("Failed to establish websocket connection: %v", err.Error())
		return nil, err
	}
	return webSocketConn, nil
}
