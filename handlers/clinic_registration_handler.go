package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/identity"
	"github.com/superdentist/superdentist-backend/lib/jwt"
	"go.opencensus.io/trace"
)

// ClinicRegistrationHandler ...
func ClinicRegistrationHandler(c *gin.Context) {
	log.Infof("Registering clinic with SD database")
	ctx := c.Request.Context()
	var clinicRegistrationReq contracts.ClinicRegistrationData
	gProjectDeployment := googleprojectlib.GetGoogleProjectID()
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
	identityClient, _ := identity.NewIDPEP(ctx, gProjectDeployment)
	currentClinicRecord, err := identityClient.GetUserByEmail(ctx, clinicRegistrationReq.EmailID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err,
			},
		)
		return
	}
	clinicDB := datastoredb.NewClinicHandler()
	err = clinicDB.InitializeDataBase(ctx, gProjectDeployment)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err,
			},
		)
		return
	}
	sdClinicID, err := clinicDB.AddClinicRegistration(ctx, &clinicRegistrationReq, currentClinicRecord.UID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err,
			},
		)
		return
	}
	responseData := contracts.ClinicRegistrationResponse{
		EmailID:    clinicRegistrationReq.EmailID,
		ClinicID:   strconv.FormatInt(sdClinicID, 10),
		IsVerified: false,
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicDB.Close()
}

// ClinicVerificationHandler ...
func ClinicVerificationHandler(c *gin.Context) {
	log.Infof("Verifying clinic with SD database")
	ctx := c.Request.Context()
	var clinicVerificationReq contracts.ClinicVerificationData
	gProjectDeployment := googleprojectlib.GetGoogleProjectID()
	userEmail, _ := jwt.GetUserEmail(c.Request)
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
	err := clinicDB.InitializeDataBase(ctx, gProjectDeployment)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err,
			},
		)
		return
	}
	identityClient, _ := identity.NewIDPEP(ctx, gProjectDeployment)
	currentClinicRecord, err := identityClient.GetUserByEmail(ctx, userEmail)
	sdClinicID, err := clinicDB.VerifyClinicInDatastore(ctx, userEmail, currentClinicRecord.UID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err,
			},
		)
		return
	}
	responseData := contracts.ClinicRegistrationResponse{
		EmailID:    userEmail,
		ClinicID:   strconv.FormatInt(sdClinicID, 10),
		IsVerified: true,
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicDB.Close()
}

// AddPhysicalClinicsHandler ... after registering clinic main account we add multiple locations etc.
func AddPhysicalClinicsHandler(c *gin.Context) {

}

// RegisterClinicDoctors .... once clinics are registers multiple doctors needs to be added to them
func RegisterClinicDoctors(c *gin.Context) {

}

// RegisterClinicPMS ..... add all PMS current clinic is using
func RegisterClinicPMS(c *gin.Context) {

}
