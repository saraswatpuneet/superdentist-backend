package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/config"
	"github.com/superdentist/superdentist-backend/global"
)

func main() {
	log.Infof("Starting superdentist backend service")

	// any global settings like PMS username/password/configuration goes here
	config.Init()
	// Only log the warning severity or above.
	if global.Options.Debug {
		log.SetLevel(log.DebugLevel)
	}
	// TODO Initialize Core that handles routes, middleware and requests
	log.Infof("Backend intialization completed")
}
