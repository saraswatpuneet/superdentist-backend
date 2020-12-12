package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/identity"
	"github.com/superdentist/superdentist-backend/lib/jwt"
	"github.com/superdentist/superdentist-backend/lib/sendgrid"
	"github.com/superdentist/superdentist-backend/lib/websocket"
	"go.opencensus.io/trace"
)

// AdminRegistrationHandler ...
func AdminRegistrationHandler(c *gin.Context) {
	log.Infof("Registering clinic with SD database")
	ctx := c.Request.Context()
	var clinicRegistrationReq contracts.ClinicRegistrationData
	_, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	log.Infof("Registering clinic with SD database2")

	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	if err := c.ShouldBindWith(&clinicRegistrationReq, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}

	clinicDB := datastoredb.NewClinicHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicDB.AddClinicRegistration(ctx, &clinicRegistrationReq, userID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	responseData := contracts.ClinicRegistrationResponse{
		EmailID:    clinicRegistrationReq.EmailID,
		IsVerified: false,
	}
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
	log.Infof("Registering clinic with SD database sendgrid")

	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	idAuth, err := identity.NewIDPEP(ctx, gproject)
	log.Infof("Registering clinic with SD database idep")

	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	veriURL, err := idAuth.GetVerificationURL(ctx, clinicRegistrationReq.EmailID)
	log.Infof("Registering clinic with SD database3")

	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = sgClient.SendVerificationEmail(clinicRegistrationReq.EmailID, veriURL)
	log.Infof("Registering clinic with SD database4")

	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicDB.Close()
}

// AdminVerificationHandler ...
func AdminVerificationHandler(c *gin.Context) {
	log.Infof("Verifying clinic with SD database")
	ctx := c.Request.Context()
	var clinicVerificationReq contracts.ClinicVerificationData
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	if err := c.ShouldBindWith(&clinicVerificationReq, binding.JSON); err != nil || !clinicVerificationReq.IsVerified {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicDB := datastoredb.NewClinicHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicDB.VerifyClinicInDatastore(ctx, userEmail, userID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	responseData := contracts.ClinicRegistrationResponse{
		EmailID:    userEmail,
		IsVerified: true,
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicDB.Close()
}

// AdminPasswordReset ...
func AdminPasswordReset(c *gin.Context) {
	log.Infof("Registering clinic with SD database")
	ctx := c.Request.Context()
	var clinicRegistrationReq contracts.PasswordResetData
	gproject := googleprojectlib.GetGoogleProjectID()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	if err := c.ShouldBindWith(&clinicRegistrationReq, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}

	sgClient := sendgrid.NewSendGridClient()
	err := sgClient.InitializeSendGridClient()
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	idAuth, err := identity.NewIDPEP(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	veriURL, err := idAuth.GetResetPasswordURL(ctx, clinicRegistrationReq.EmailID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = sgClient.SendPasswordResetEmail(clinicRegistrationReq.EmailID, veriURL)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "password reset in progress",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// AddPhysicalClinicsHandler ... after registering clinic main account we add multiple locations etc.
func AddPhysicalClinicsHandler(c *gin.Context) {
	log.Infof("Adding physical addresses to database for logged in clinic")
	ctx := c.Request.Context()
	var addClinicAddressRequest contracts.PostPhysicalClinicDetails
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Register address for various clinics for this admin")
	mapClient := gmaps.NewMapsHandler()
	err = mapClient.InitializeGoogleMapsAPIClient(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	defer span.End()
	if err = c.ShouldBindWith(&addClinicAddressRequest, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	registeredClinics, err := clinicMetaDB.AddPhysicalAddessressToClinic(ctx, userEmail, userID, addClinicAddressRequest.ClinicDetails, mapClient)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	responseData := contracts.ClinicAddressResponse{
		ClinicDetails: registeredClinics,
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// RegisterClinicDoctors .... once clinics are registers multiple doctors needs to be added to them
func RegisterClinicDoctors(c *gin.Context) {
	log.Infof("Adding doctors to clinics identified by their addressId")
	ctx := c.Request.Context()
	var addClinicAddressRequest contracts.PostDoctorDetails
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Register doctors for various clinics for this admin")
	defer span.End()
	if err := c.ShouldBindWith(&addClinicAddressRequest, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicMetaDB.AddDoctorsToPhysicalClincs(ctx, userEmail, userID, addClinicAddressRequest.Doctors)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "Doctors have been successfully registered",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// RegisterClinicPMS ..... add all PMS current clinic is using
func RegisterClinicPMS(c *gin.Context) {
	log.Infof("Adding PMS list used by clinics")
	ctx := c.Request.Context()
	var addPMSList contracts.PostPMSDetails
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Register PMS list used by clinics")
	defer span.End()
	if err := c.ShouldBindWith(&addPMSList, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicMetaDB.AddPMSUsedByClinics(ctx, userEmail, userID, addPMSList.PMSNames)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "PMS have registered for the clinic",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// AddPMSAuthDetails ..... all authorization details for PMS
func AddPMSAuthDetails(c *gin.Context) {
	log.Infof("Adding PMS list used by clinics")
	ctx := c.Request.Context()
	var addPMSAuth contracts.PostPMSAuthDetails
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Register PMS list used by clinics")
	defer span.End()
	if err := c.ShouldBindWith(&addPMSAuth, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicMetaDB.AddPMSAuthDetails(ctx, userEmail, userID, addPMSAuth)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "PMS have registered for the clinic",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// RegisterSpecialityServices .... register any special services a clinic admin offers
func RegisterSpecialityServices(c *gin.Context) {
	log.Infof("Adding services offered by clinics")
	ctx := c.Request.Context()
	var addServices contracts.PostClinicServices
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Adding services offered by clinics")
	defer span.End()
	if err := c.ShouldBindWith(&addServices, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	err = clinicMetaDB.AddServicesForClinic(ctx, userEmail, userID, addServices.Services)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "Services have registered for the clinic",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetAddressListRest ...
func GetAddressListRest(c *gin.Context) {
	ctx := c.Request.Context()
	gproject := googleprojectlib.GetGoogleProjectID()
	searchText := c.Query("searchText")

	mapClient := gmaps.NewMapsHandler()
	err := mapClient.InitializeGoogleMapsAPIClient(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	resultPlaces, err := mapClient.FindPlacesFromText(searchText)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	restunedResults := contracts.PostAddressList{
		AddressList: resultPlaces.Results,
		Error:       "",
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   restunedResults,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// QueryAddressHandlerWebsocket ...
func QueryAddressHandlerWebsocket(webPool *websocket.Pool, c *gin.Context) {
	ctx := c.Request.Context()
	gproject := googleprojectlib.GetGoogleProjectID()

	mapClient := gmaps.NewMapsHandler()
	err := mapClient.InitializeGoogleMapsAPIClient(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	webSocketConn, err := websocket.UpgradeWebSocket(c)
	if err != nil {
		log.Errorf("Failed to establish websocket connection: %v", err.Error())
	}
	connID, _ := uuid.NewUUID()
	client := &websocket.Client{
		CurrentPool:   webPool,
		CurrentConn:   webSocketConn,
		CurrentConnID: connID.String(),
		Send:          make(chan []byte, 1024),
	}
	client.CurrentPool.Register <- &websocket.RegisterChannel{
		ClientID:  connID.String(),
		WebClient: client,
	}
	go client.ReadAddressString()
	go client.WriteAdderessJSON(mapClient)
}
func getUserDetails(ctx context.Context, request *http.Request) (string, string, string, error) {
	gProjectDeployment := googleprojectlib.GetGoogleProjectID()
	identityClient, err := identity.NewIDPEP(ctx, gProjectDeployment)
	if err != nil {
		log.Errorf("found error in identity: %v", err.Error())
		return "", "", "", err
	}
	userEmail, _ := jwt.GetUserEmail(request)
	currentClinicRecord, err := identityClient.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return "", "", "", err
	}
	if userEmail != currentClinicRecord.Email {
		return "", "", "", fmt.Errorf("Unauthorized access: aborting")
	}
	return currentClinicRecord.Email, currentClinicRecord.UID, gProjectDeployment, nil
}
