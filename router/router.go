package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/handlers"
	"github.com/superdentist/superdentist-backend/lib/websocket"
)

// SDRouter ... superdentist backend router to handle various APIs
func SDRouter() (*gin.Engine, error) {
	// Initialize and run websocket pool manager
	poolConnections := websocket.NewPool()
	go poolConnections.RunPool()
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
		clinicGroup.POST("/directJoin", handlers.DirectJoinHandler)
		clinicGroup.PUT("/passwordReset", handlers.AdminPasswordReset)
		clinicGroup.POST("/addClinics", handlers.AddPhysicalClinicsHandler)
		clinicGroup.POST("/registerDoctors", handlers.RegisterClinicDoctors)
		clinicGroup.POST("/registerPMS", handlers.RegisterClinicPMS)
		clinicGroup.POST("/registerPMSAuth", handlers.AddPMSAuthDetails)
		clinicGroup.POST("/registerServices", handlers.RegisterSpecialityServices)
	}
	{
		// All data query related APIs: Basic stuff R
		clinicGroup.GET("/getClinics", handlers.GetPhysicalClinics)
		clinicGroup.GET("/getAll", handlers.GetAllClinicNameAddressID)
		clinicGroup.GET("/getDoctors/:addressId", handlers.GetClinicDoctors)
		clinicGroup.GET("/getAllDoctors", handlers.GetAllDoctors)
		clinicGroup.POST("/getNearbySpecialists", handlers.GetNearbySpeialists)
		clinicGroup.POST("/addFavorites/:addressId", handlers.AddFavoriteClinics)
		clinicGroup.GET("/qrimages/:placeId", handlers.GetAllQRZip)
		clinicGroup.GET("/getFavorites/:addressId", handlers.GetFavoriteClinics)
		clinicGroup.GET("/getNetwork/:addressId", handlers.GetNetworkClinics)
		clinicGroup.POST("/removeFavorites/:addressId", handlers.RemoveFavoriteClinics)
		clinicGroup.POST("/practiceCodes/:addressId", handlers.AddClinicPracticeCodes)
		clinicGroup.GET("/practiceCodes/:addressId", handlers.GetClinicPracticeCodes)
		clinicGroup.POST("/practiceCodesHistory/:addressId", handlers.AddClinicPracticeCodesHistory)
		clinicGroup.GET("/practiceCodesHistory/:addressId", handlers.GetClinicPracticeCodesHistory)
	}
	referralGroup := version1.Group("/")
	{
		referralGroup.POST("/referral/mail", handlers.ReceiveReferralMail)
		referralGroup.POST("/summary/mail", handlers.ReceiveAutoSummaryMail)
		referralGroup.POST("/referral/scheduledemo", handlers.ScheduleDemo)
		referralGroup.POST("/referral/sms", handlers.TextRecievedPatient)
		referralGroup.POST("/referrals", handlers.CreateRefSpecialist)
		referralGroup.POST("/qrReferral", handlers.QRReferral)
		referralGroup.POST("/referrals/:referralId/messages", handlers.AddCommentsToReferral)
		referralGroup.GET("/referrals/:referralId/messages", handlers.GetAllMessages)
		referralGroup.GET("/referrals/:referralId/messages/:messageId", handlers.GetOneMessage)
		referralGroup.PUT("/referrals/:referralId/status", handlers.UpdateReferralStatus)
		referralGroup.DELETE("/referrals/:referralId", handlers.DeleteReferral)
		referralGroup.POST("/referrals/:referralId/documents", handlers.UploadDocuments)
		referralGroup.GET("/referrals/:referralId/documents", handlers.DownloadDocumentsAsZip)
		referralGroup.GET("/referrals/:referralId/document", handlers.DownloadSingleFile)
		referralGroup.GET("/referrals-by-clinic/dentist", handlers.GetAllReferralsGD)
		referralGroup.GET("/referrals-by-clinic/specialist", handlers.GetAllReferralsSP)
		referralGroup.GET("/referrals/:referralId", handlers.GetOneReferral)

	}
	adminGroup := version1.Group("/admin")
	{
		// All data entry related APIs: Basic Stuff C & U
		adminGroup.POST("/addFavorites/:addressId", handlers.AddFavoriteClinics)

	}
	patientGroup := version1.Group("/patient")
	{
		patientGroup.POST("/registration", handlers.RegisterPatientInformation)
		patientGroup.GET("/list/:addressId", handlers.GetAllPatientsForClinic)
		patientGroup.POST("/notes/:patientId", handlers.AddPatientNotes)
		patientGroup.POST("/status/:patientId", handlers.AddPatientNotes)
		patientGroup.GET("/notes/:patientId", handlers.GetPatientNotes)
		patientGroup.POST("/files/:patientId", handlers.UploadPatientDocuments)
		patientGroup.POST("/processSheet", handlers.ProcessPatientSpreadSheet)

	}
	insuranceGroup := version1.Group("/insurance")
	{
		insuranceGroup.GET("/practiceCodes", handlers.GetAllPracticeCodesCats)
		insuranceGroup.GET("/dentalInsurance", handlers.GetAllDentalInsurances)
		insuranceGroup.GET("/medicalInsurance", handlers.GetAllMedicalInsurances)

	}
	{
		// All wesocket related routing goes here follow the pattern

		clinicGroup.GET("/queryAddress", func(c *gin.Context) {
			handlers.QueryAddressHandlerWebsocket(poolConnections, c)
		})

		clinicGroup.GET("/getAddressList", handlers.GetAddressListRest)
	}
	// Derive groups from version group to consolidate our APIs in a better way
	return restRouter, nil
}
