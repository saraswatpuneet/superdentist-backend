package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/router"
	graceful "gopkg.in/tylerb/graceful.v1" // see: https://github.com/tylerb/graceful
)

type maxPayloadHandler struct {
	handler http.Handler
	size    int64
}

// ServeHTTP uses MaxByteReader to limit the size of the input
func (handler *maxPayloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, handler.size)
	handler.handler.ServeHTTP(w, r)
}

// SDBackendController ....
func SDBackendController(ctx context.Context, port int, errorChannel chan error) {
	log.Infof("Initializing router and endpoints.")
	sdRouter, err := router.SDRouter()
	if err != nil {
		errorChannel <- err
		return
	}
	httpAddress := fmt.Sprintf(":%d", port)
	// set maximum payload size for incoming requests

	// init maxHandler to limit size of request input
	var httpHandler http.Handler
	if global.Options.MaxPayloadSize > 0 {
		httpHandler = &maxPayloadHandler{handler: sdRouter, size: global.Options.MaxPayloadSize}
	} else {
		httpHandler = sdRouter
	}
	global.WaitGroupServer.Add(1)
	go serverHTTPRoutes(ctx, httpAddress, httpHandler, errorChannel)
}

func serverHTTPRoutes(ctx context.Context, httpAddress string, handler http.Handler, errorChannel <-chan error) {
	defer global.WaitGroupServer.Done()
	// init graceful server
	serverGrace := &graceful.Server{
		Timeout: 10 * time.Second,
		//BeforeShutdown:    beforeShutDown,
		ShutdownInitiated: shutDownBackend,
		Server: &http.Server{
			Addr:           httpAddress,
			Handler:        handler,
			MaxHeaderBytes: global.Options.MaxHeaderSize,
			ReadTimeout:    time.Duration(constants.MAX_READ_TIMEOUT) * time.Second,
			WriteTimeout:   time.Duration(constants.MAX_WRITE_TIMEOUT) * time.Second,
		},
	}
	stopChannel := serverGrace.StopChan()
	err := serverGrace.ListenAndServe()
	if err != nil {
		log.Fatalf("SDController: Failed to start server : %s", err.Error())
	}
	log.Infof("Backend is serving the routes.")
	for {
		// wait for the server to stop or be canceled
		select {
		case <-stopChannel:
			log.Infof("SDController: Server shutdown at %s", time.Now())
			return
		case <-ctx.Done():
			log.Infof("SDController: context done is called %s", time.Now())
			serverGrace.Stop(time.Second * 2)
		}
	}
}

func shutDownBackend() {
	log.Infof("SDController: Shutting down server at %s", time.Now())
}
