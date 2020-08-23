package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/handlers"
)

// SDRouter ... superdentist backend router to handle various APIs
func SDRouter() (*gin.Engine, error) {
	restRouter := gin.Default()
	// configure cors as needed for FE/BE interactions: For now defaults

	configCors := cors.DefaultConfig()
	configCors.AllowAllOrigins = true
	configCors.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	restRouter.Use(cors.New(configCors))

	// TODO: inti rout handlers

	//
	if !global.Options.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// This router is not added to the v1 group even though the route prefix is the same to avoid
	// making it require authentication (k8s need it for liveness check)
	restRouter.GET("/api/v1/healthcheck", handlers.HealthCheckHandler)

	// TODO: add any future routes here

	//
	return restRouter, nil
}

