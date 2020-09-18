package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/config"
	"github.com/superdentist/superdentist-backend/global"
	servertrigger "github.com/superdentist/superdentist-backend/trigger"
)

func main() {
	log.Infof("Starting superdentist backend service")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "D:\\SuperDentist\\keys\\super-dentist-backend.json")
	// any global settings like PMS username/password/configuration goes here
	config.Init()
	// Only log the warning severity or above.
	if global.Options.Debug {
		log.SetLevel(log.DebugLevel)
	}
	// Initialize Rest APIs
	err := servertrigger.CoreServer()
	if err != nil {
		//send signal to all channels to calm down we found an error
		log.Errorf("Backend server went crazy: %v", err.Error())
		// send OS signal and shut it all down
		os.Exit(1)
	}
	log.Infof("Backend intialization completed")
}
