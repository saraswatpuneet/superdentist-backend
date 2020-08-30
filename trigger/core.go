// Package trigger entry point of go service
package trigger

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/superdentist/superdentist-backend/controller"
	"github.com/superdentist/superdentist-backend/global"
)

// CoreServer ....CoreServer
func CoreServer() error {
	// create a new context and save it in global
	port := 8080
	ctx, cancel := context.WithCancel(context.Background())
	log.Infof("Starting superdentist-backend container.")
	portFromEnv := os.Getenv("PORT")
	var err error
	if portFromEnv != "" {
		port, err = strconv.Atoi(portFromEnv)
		if err != nil {
			port = 8090
		}
	}
	global.Ctx = ctx

	// setup cancel signal for graceful shutdown of serve\
	go monitorSystem(cancel)
	bootUPErrors := make(chan error, 1)

	controller.SDBackendController(ctx, port, bootUPErrors)
	//........................................................................................
	// Block the server until server encounter any errors
	err = <-bootUPErrors
	if err != nil {
		log.Errorf("There is an issue starting backend server for super dentist: %v", err.Error())
		global.WaitGroupServer.Wait()
		return err
	}
	log.Infof("SuperDentist backend server started with side car proxy.")
	return nil
}

func monitorSystem(cancel context.CancelFunc) {
	holdSignal := make(chan os.Signal, 1)
	signal.Notify(holdSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	// if system throw any termination stuff let channel handle it and cancel
	<-holdSignal
	cancel()
}
