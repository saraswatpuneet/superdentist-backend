package handlers

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

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
