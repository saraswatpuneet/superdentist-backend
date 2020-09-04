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

	// TODO: inti route handlers

	//
	if !global.Options.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	//.....................................................................
	// healthcheck is need by Kubernetes to test readiness of containers
	// register route is again not protected since it will be used for registration
	// todo prevent spam/bot attaches for register route
	// login route will take in user info check against IAP/IP and return token/reject
	restRouter.GET("/", handlers.HealthCheckHandler)
	restRouter.GET("/healthz", handlers.HealthCheckHandler)
	restRouter.GET("/api/v1/healthz", handlers.HealthCheckHandler)
	// ....................................................................
	// TODO: add any future routes here
	restRouter.Static("/docs/", "./swaggerui")

	//
	return restRouter, nil
}
