package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// HelloHandler ...
func HelloHandler(c *gin.Context) {
	log.Infof("Hello handler called.")
	healthStatus := "Super Dentist Says Hi"
	response, _ := json.Marshal(healthStatus)
	c.Writer.Header().Add("content-type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)

	if _, err := c.Writer.Write(response); err != nil {
		log.Errorf("GetHello ... something went wrong with APIs: %v", err)
	}

}

// UserRegistrationHandler ...
func UserRegistrationHandler(c *gin.Context) {
	log.Infof("Signing up user to platform")
	// TODO: implement GCP indentity platform registration routine

}

// UserLoginHandler ...
func UserLoginHandler(c *gin.Context) {
	log.Infof("Attempting to login user.")
	// TODO: implement GCP authentication checks and return tokens back
}
