package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"googlemaps.github.io/maps"
)

// SendRefereshSignal ... write json places back to socket for FE to provide suggestions
func (c *Client) SendRefereshSignal(mapClient *gmaps.ClientGMaps) {
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
