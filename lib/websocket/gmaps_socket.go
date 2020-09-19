package websocket

import (
	"bytes"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"googlemaps.github.io/maps"
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
		messageType, message, err := c.CurrentConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		if messageType != websocket.TextMessage {
			currentBlankResponse := maps.FindPlaceFromTextResponse{}
			returnError := contracts.PostAddressList{
				AddressList: currentBlankResponse,
				Error:       "Websocket only accepts text message",
			}
			c.CurrentConn.WriteJSON(returnError)
			c.CurrentConn.Close()
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.CurrentPool.Broadcast <- message
	}
}

// WriteAdderessJSON ... write json places back to socket for FE to provide suggestions
func (c *Client) WriteAdderessJSON(mapClient *gmaps.ClientGMaps) {
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
				// The pool closed the channel.
				c.CurrentConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			resultPlaces, err := mapClient.FindPlacesFromText(string(message))
			if err != nil {
				return
			}
			err = c.CurrentConn.WriteJSON(resultPlaces)
			if err != nil {
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
