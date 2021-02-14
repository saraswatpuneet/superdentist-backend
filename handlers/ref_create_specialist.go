package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/otiai10/gosseract/v2"
	"gopkg.in/ugjka/go-tz.v2/tz"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
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
	processReferral(ctx, c, referralDetails, gproject)
}

// QRReferral ...
func QRReferral(c *gin.Context) {
	ctx := c.Request.Context()
	log.Infof("Received QR referral request")
	secureKey := c.Query("secureKey")
	patientFName := c.Query("firstName")
	patientLName := c.Query("lastName")
	patientPhone := c.Query("phone")
	patientEmail := c.Query("email")
	from := c.Query("from")
	to := c.Query("to")
	decryptedKey, err := decrypt(secureKey)
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
	splitKey := strings.Split(decryptedKey, "+")
	logo := splitKey[0]
	numeric := splitKey[1]
	boolean := splitKey[2]
	if numeric != "10074" && boolean != "true" && logo != "superdentist" {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	gproject := googleprojectlib.GetGoogleProjectID()
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
	fromClinic := from
	toClinic := to
	var referralDetails contracts.ReferralDetails
	referralDetails.FromPlaceID = fromClinic
	referralDetails.ToPlaceID = toClinic
	referralDetails.Patient = contracts.Patient{
		FirstName: patientFName,
		LastName:  patientLName,
		Phone:     patientPhone,
		Email:     patientEmail,
	}
	referralDetails.Status.GDStatus = "referred"
	referralDetails.Status.SPStatus = "referred"
	referral, updatedComm := processReferral(ctx, c, referralDetails, gproject)
	var refComments contracts.ReferralComments
	refComments.Comments = append(refComments.Comments, updatedComm.Comments...)
	_, err = ProcessComments(ctx, gproject, referral.ReferralID, refComments)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   nil,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

func processReferral(ctx context.Context, c *gin.Context, referralDetails contracts.ReferralDetails, gproject string) (*contracts.DSReferral, *contracts.ReferralComments) {
	storageC := storage.NewStorageHandler()
	err := storageC.InitializeStorageClient(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return nil, nil
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
		return nil, nil
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
		return nil, nil
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
		return nil, nil
	}
	currentRefUUID, _ := uuid.NewUUID()
	uniqueRefID := currentRefUUID.String()
	docIDNames := make([]string, 0)
	// Stage 2 Upload files from
	// parse request
	client := gosseract.NewClient()
	defer client.Close()
	foundImage := false
	const _24K = (1 << 10) * 100
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
						return nil, nil
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
					return nil, nil
				}
				imageBuffer := bytes.NewBuffer(nil)
				if _, err := io.Copy(imageBuffer, infile); err != nil {
					c.AbortWithStatusJSON(
						http.StatusInternalServerError,
						gin.H{
							constants.RESPONSE_JSON_DATA:   nil,
							constants.RESPONSDE_JSON_ERROR: err.Error(),
						},
					)
					return nil, nil
				}
				client.SetImageFromBytes(imageBuffer.Bytes())
				foundImage = true
				io.Copy(buckerW, infile)
				buckerW.Close()
				docIDNames = append(docIDNames, hdr.Filename)
			}
		}
		err = storageC.ZipFile(ctx, uniqueRefID, constants.SD_REFERRAL_BUCKET)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					constants.RESPONSE_JSON_DATA:   nil,
					constants.RESPONSDE_JSON_ERROR: err.Error(),
				},
			)
			return nil, nil
		}
	}
	ocrText := ""
	if foundImage {
		ocrText, _ = client.Text()
	}
	var dsReferral contracts.DSReferral
	if ocrText != "" {
		startIndex := strings.Index(ocrText, "Reason")
		endIndex := strings.Index(ocrText, "Faster")

		if startIndex >= 0 && endIndex-1 > 0 {
			ocrText = ocrText[startIndex:endIndex-1]
		} else if startIndex >= 0 {
			ocrText = ocrText[startIndex:]
		}
		dsReferral.Reasons = []string{ocrText}
	}
	if referralDetails.Patient.Phone != "" {
		currentPhone := referralDetails.Patient.Phone
		pnum, _ := phonenumbers.Parse(currentPhone, "US")
		countryCode := "+1"
		if pnum.CountryCode != nil {
			countryCode = "+" + strconv.Itoa(int(*pnum.CountryCode))
		}
		dsReferral.PatientPhone = countryCode + strconv.Itoa(int(*pnum.NationalNumber))
	}

	dsReferral.Documents = docIDNames
	dsReferral.CreatedOn = time.Now()
	dsReferral.ModifiedOn = time.Now()
	dsReferral.ReferralID = uniqueRefID
	dsReferral.Reasons = referralDetails.Reasons
	dsReferral.Status = referralDetails.Status
	dsReferral.History = referralDetails.History
	updatedComm := make([]contracts.Comment, 0)
	for _, comm := range referralDetails.Comments {
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
	if referralDetails.FromAddressID != "" {
		fromClinic, err := clinicDB.GetSingleClinic(ctx, referralDetails.FromAddressID)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					constants.RESPONSE_JSON_DATA:   nil,
					constants.RESPONSDE_JSON_ERROR: err.Error(),
				},
			)
			return nil, nil
		}
		dsReferral.FromPlaceID = fromClinic.PlaceID
		dsReferral.FromClinicName = fromClinic.Name
		dsReferral.FromClinicAddress = fromClinic.Address
		dsReferral.FromEmail = fromClinic.EmailAddress
		dsReferral.FromClinicPhone = fromClinic.PhoneNumber
	} else {
		fromClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, referralDetails.FromPlaceID)

		if err == nil && fromClinic.PhysicalClinicsRegistration.Name != "" {
			dsReferral.FromPlaceID = fromClinic.PlaceID
			dsReferral.FromClinicName = fromClinic.Name
			dsReferral.FromAddressID = fromClinic.AddressID
			dsReferral.FromClinicAddress = fromClinic.Address
			dsReferral.FromEmail = fromClinic.EmailAddress
			dsReferral.FromClinicPhone = fromClinic.PhoneNumber
			zone, _ := tz.GetZone(tz.Point{
				Lon: fromClinic.Location.Long, Lat: fromClinic.Location.Lat,
			})
			var commentReasons contracts.Comment
			currentID, _ := uuid.NewUUID()
			commentReasons.MessageID = currentID.String()
			commentReasons.Channel = "c2c"
			commentReasons.Text = "New QR based referral is created."
			location, _ := time.LoadLocation(zone[0])
			commentReasons.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
			commentReasons.UserID = fromClinic.EmailAddress
			updatedComm = append(updatedComm, commentReasons)
			if ocrText != "" {
				var commentReasons contracts.Comment
				currentID, _ := uuid.NewUUID()
				commentReasons.MessageID = currentID.String()
				commentReasons.Channel = "c2c"
				commentReasons.Text = ocrText
				location, _ := time.LoadLocation(zone[0])
				commentReasons.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
				commentReasons.UserID = fromClinic.EmailAddress
				updatedComm = append(updatedComm, commentReasons)
			}
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
			details, _ := mapClient.FindPlaceFromID(referralDetails.FromPlaceID)
			dsReferral.FromPlaceID = referralDetails.FromPlaceID
			if details != nil && details.PlaceID == referralDetails.FromPlaceID {
				dsReferral.FromClinicName = details.Name
				dsReferral.FromClinicAddress = details.FormattedAddress
				dsReferral.FromClinicPhone = details.FormattedPhoneNumber
			}
		}
	}
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
			return nil, nil
		}
		dsReferral.ToPlaceID = toClinic.PlaceID
		dsReferral.ToClinicName = toClinic.Name
		dsReferral.ToClinicAddress = toClinic.Address
		dsReferral.ToEmail = toClinic.EmailAddress
		dsReferral.ToClinicPhone = toClinic.PhoneNumber
		dsReferral.CommunicationPhone = toClinic.TwilioNumber
		dsReferral.CommunicationText = toClinic.CustomText

	} else {
		toClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, referralDetails.ToPlaceID)
		if err == nil && toClinic.PhysicalClinicsRegistration.Name != "" {
			dsReferral.ToPlaceID = toClinic.PlaceID
			dsReferral.ToClinicName = toClinic.Name
			dsReferral.ToClinicAddress = toClinic.Address
			dsReferral.ToEmail = toClinic.EmailAddress
			dsReferral.ToClinicPhone = toClinic.PhoneNumber
			dsReferral.ToAddressID = toClinic.AddressID
			dsReferral.CommunicationPhone = toClinic.TwilioNumber
			dsReferral.CommunicationText = toClinic.CustomText
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
		return nil, nil
	}
	var returnComments contracts.ReferralComments
	returnComments.Comments = updatedComm
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	return &dsReferral, &returnComments
}

func decrypt(ciphertext64 string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(ciphertext64)
	if err != nil {
		return "", err
	}

	nonceSize := global.Options.GCMQR.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	b, err := global.Options.GCMQR.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(b), err
}
