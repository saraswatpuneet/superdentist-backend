package websocket

import (
	"bytes"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// ReadAddressString ....
func (c *Client) ReadAddressString() {
	defer func() {
		c.CurrentPool.Unregister <- c
		c.CurrentConn.Close()
	}()
	c.CurrentConn.SetReadLimit(MaxMessageSize)
	c.CurrentConn.SetReadDeadline(time.Now().Add(PongWait))
	c.CurrentConn.SetPongHandler(func(string) error { c.CurrentConn.SetReadDeadline(time.Now().Add(PongWait)); return nil })
	for {
		_, message, err := c.CurrentConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.CurrentPool.Broadcast <- message
	}
}

// WriteAdderessJSON ... write json places back to socket for FE to provide suggestions
func (c *Client) WriteAdderessJSON() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.CurrentConn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.CurrentConn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// The hub closed the channel.
				c.CurrentConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.CurrentConn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.CurrentConn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.CurrentConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
