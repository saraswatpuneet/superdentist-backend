package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/sendgrid"
	"github.com/superdentist/superdentist-backend/lib/storage"
	"go.opencensus.io/trace"
)

// CreateRefSpecialist ...
func CreateRefSpecialist(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Creating Referral")
	ctx := c.Request.Context()
	var referralDetails contracts.ReferralDetails
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	if err := c.ShouldBindWith(&referralDetails, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	storageC := storage.NewStorageHandler()
	err = storageC.InitializeStorageClient(ctx, gproject)
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
	clinicDB := datastoredb.NewClinicMetaHandler()
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
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	currentRefUUID, _ := uuid.NewUUID()
	uniqueRefID := currentRefUUID.String()
	docIDNames := make([]string, 0)
	// Stage 2 Upload files from
	// parse request
	const _24K = (1 << 10) * 24
	if err = c.Request.ParseMultipartForm(_24K); err == nil {
		for _, fheaders := range c.Request.MultipartForm.File {
			for _, hdr := range fheaders {
				// open uploaded
				var infile multipart.File
				if infile, err = hdr.Open(); err != nil {
					if err := c.ShouldBindWith(&referralDetails, binding.JSON); err != nil {
						c.AbortWithStatusJSON(
							http.StatusBadRequest,
							gin.H{
								constants.RESPONSE_JSON_DATA:   nil,
								constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad files sent to backend"),
							},
						)
						return
					}
				}
				fileName := hdr.Filename
				bucketPath := uniqueRefID + "/" + fileName
				buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
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
				io.Copy(buckerW, infile)
				buckerW.Close()
				docIDNames = append(docIDNames, hdr.Filename)
			}
		}
		err = storageC.ZipFile(ctx, uniqueRefID)
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
	}
	var dsReferral contracts.DSReferral
	currentPhone := referralDetails.Patient.Phone
	pnum, _ := phonenumbers.Parse(currentPhone, "US")
	countryCode := "+1"
	if pnum.CountryCode != nil {
		countryCode = "+" + strconv.Itoa(int(*pnum.CountryCode))
	}
	dsReferral.PatientPhone = countryCode + strconv.Itoa(int(*pnum.NationalNumber))

	dsReferral.Documents = docIDNames
	dsReferral.CreatedOn = time.Now()
	dsReferral.ModifiedOn = time.Now()
	dsReferral.ReferralID = uniqueRefID
	dsReferral.Reasons = referralDetails.Reasons
	dsReferral.Status = referralDetails.Status
	dsReferral.History = referralDetails.History
	updatedComm := make([]contracts.Comment, 0)
	for _, comm := range referralDetails.Comments {
		comm.TimeStamp = time.Now().Unix()
		currentID, _ := uuid.NewUUID()
		comm.MessageID = currentID.String()
		updatedComm = append(updatedComm, comm)
	}
	dsReferral.Tooth = referralDetails.Tooth
	dsReferral.PatientEmail = referralDetails.Patient.Email
	dsReferral.PatientFirstName = referralDetails.Patient.FirstName
	dsReferral.PatientLastName = referralDetails.Patient.LastName
	dsReferral.IsDirty = false
	dsReferral.FromAddressID = referralDetails.FromAddressID
	dsReferral.ToAddressID = referralDetails.ToAddressID
	// Stage 3 Create datastore entry for referral
	fromClinic, err := clinicDB.GetSingleClinic(ctx, referralDetails.FromAddressID)
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
	dsReferral.FromPlaceID = fromClinic.PlaceID
	dsReferral.FromClinicName = fromClinic.Name
	dsReferral.FromClinicAddress = fromClinic.Address
	dsReferral.FromEmail = fromClinic.EmailAddress
	dsReferral.FromClinicPhone = fromClinic.PhoneNumber
	if referralDetails.ToAddressID != "" {
		toClinic, err := clinicDB.GetSingleClinic(ctx, referralDetails.ToAddressID)
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
		dsReferral.ToPlaceID = toClinic.PlaceID
		dsReferral.ToClinicName = toClinic.Name
		dsReferral.ToClinicAddress = toClinic.Address
		dsReferral.ToEmail = toClinic.EmailAddress
		dsReferral.ToClinicPhone = toClinic.PhoneNumber
	} else {
		toClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, referralDetails.ToPlaceID)
		if err == nil && toClinic.IsVerified {
			dsReferral.ToPlaceID = toClinic.PlaceID
			dsReferral.ToClinicName = toClinic.Name
			dsReferral.ToClinicAddress = toClinic.Address
			dsReferral.ToEmail = toClinic.EmailAddress
			dsReferral.ToClinicPhone = toClinic.PhoneNumber
		} else {
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
			}
			details, err := mapClient.FindPlaceFromID(referralDetails.ToPlaceID)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						constants.RESPONSE_JSON_DATA:   nil,
						constants.RESPONSDE_JSON_ERROR: err.Error(),
					},
				)
			}
			dsReferral.ToClinicAddress = details.FormattedAddress
			dsReferral.ToPlaceID = details.PlaceID
			dsReferral.ToClinicName = details.Name
			dsReferral.ToClinicPhone = details.FormattedPhoneNumber
		}
	}
	dsReferral.IsNew = true
	err = dsRefC.CreateReferral(ctx, dsReferral)
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
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}
