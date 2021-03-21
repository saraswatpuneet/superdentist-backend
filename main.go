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
	// Only for local debugging and testing ......
	//os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "D:\\SuperDentist\\keys\\super-dentist-backend.json")
	//os.Setenv("GCP_API_KEY", "AIzaSyCp-tO9Rk5iWTeg-bqtP2tvFaW9dXlsS6k")
	//os.Setenv("TWI_SID", "AC43986acdc81f461768b8dd1fb0e17f3d")
	//os.Setenv("TWI_AUTH", "fbec2dcbf1d93d233b34b1d39dabf064")
	//os.Setenv("SENDGRID_API_KEY", "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")
	// ...........................................

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
