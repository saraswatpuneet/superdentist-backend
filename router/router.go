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
	restRouter.GET("/healthz", handlers.HealthCheckHandler)
	version1 := restRouter.Group("/v1")

	//.....................................................................
	// healthcheck is need by Kubernetes to test readiness of containers
	// register route is again not protected since it will be used for registration
	// todo prevent spam/bot attaches for register route
	// login route will take in user info check against IAP/IP and return token/reject
	clinicGroup := version1.Group("/clinic")
	{
		// All data entry related APIs: Basic Stuff C & U
		clinicGroup.POST("/registerAdmin", handlers.AdminRegistrationHandler)
		clinicGroup.POST("/verifyAdmin", handlers.AdminVerificationHandler)
		clinicGroup.POST("/addClinics", handlers.AddPhysicalClinicsHandler)
		clinicGroup.POST("/registerDoctors", handlers.RegisterClinicDoctors)
		clinicGroup.POST("/registerPMS", handlers.RegisterClinicPMS)
		clinicGroup.POST("/registerServices", handlers.RegisterSpecialityServices)
	}
	{
		// All data query related APIs: Basic stuff R
	}
	// Derive groups from version group to consolidate our APIs in a better way
	return restRouter, nil
}
