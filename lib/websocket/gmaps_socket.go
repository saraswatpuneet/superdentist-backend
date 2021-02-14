package websocket

import (
	"bytes"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"googlemaps.github.io/maps"
)

// ReadAddressString ....
func (c *Client) ReadAddressString() {
	defer func() {
		c.CurrentPool.Unregister <- &UnRegisterChannel{
			ClientID:  c.CurrentConnID,
			WebClient: c,
		}
		c.CurrentConn.Close()
	}()
	c.CurrentConn.SetReadLimit(MaxMessageSize)
	c.CurrentConn.SetReadDeadline(time.Now().Add(PongWait))
	c.CurrentConn.SetPongHandler(func(string) error { c.CurrentConn.SetReadDeadline(time.Now().Add(PongWait)); return nil })
	for {
		messageType, message, err := c.CurrentConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}
		if messageType != websocket.TextMessage {
			currentBlankResponse := make([]maps.PlacesSearchResult, 0)
			returnError := contracts.PostAddressList{
				AddressList: currentBlankResponse,
				Error:       "Websocket only accepts text message",
			}
			log.Errorf("bad message sent to backend: %v", string(message))
			c.CurrentConn.WriteJSON(returnError)
			c.CurrentConn.Close()
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.CurrentPool.Broadcast <- BroadCastChannel{
			ClientID: c.CurrentConnID,
			Message:  message,
		}

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
				log.Errorf("error finding places: %v", err.Error())
				currentBlankResponse := make([]maps.PlacesSearchResult, 0)
				returnError := contracts.PostAddressList{
					AddressList: currentBlankResponse,
					Error:       err.Error(),
				}
				c.CurrentConn.WriteJSON(returnError)
				c.CurrentConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			restunedResults := contracts.PostAddressList{
				AddressList: resultPlaces.Results,
				Error:       "",
			}
			err = c.CurrentConn.WriteJSON(restunedResults)
			if err != nil {
				log.Errorf("error writing places: %v", err.Error())
				c.CurrentConn.WriteMessage(websocket.CloseMessage, []byte{})
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

// ReadChatPayload ...
func(c *Client) ReadChatPayload(){

}

// WritePingsToChat ...
func(c *Client) WritePingsToChat(){

}
