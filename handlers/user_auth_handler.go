package handlers

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// UserRegistrationHandler ...
func UserRegistrationHandler(c *gin.Context) {
	log.Infof("Registering Signed Up User to SD database")

}