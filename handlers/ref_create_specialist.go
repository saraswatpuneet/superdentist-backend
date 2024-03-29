package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ugjka/go-tz.v2/tz"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/helpers"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/identity"
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
	const _24K = 256 << 20
	var documentFiles *multipart.Form
	if err = c.Request.ParseMultipartForm(_24K); err == nil {
		documentFiles = c.Request.MultipartForm
	}
	dsReferral, _ := processReferral(referralDetails, gproject, false, documentFiles)
	if dsReferral == nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: "Unable to create referral",
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
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
	dobYear := c.Query("year")
	dobMonth := c.Query("month")
	dobDay := c.Query("day")
	from := c.Query("from")
	to := c.Query("to")
	decryptedKey, err := helpers.DecryptAndDecode(secureKey)
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
	referralDetails.IsSummary = false
	referralDetails.Patient = contracts.PatientStore{
		FirstName: patientFName,
		LastName:  patientLName,
		Phone:     patientPhone,
		Email:     patientEmail,
		Dob: contracts.DOB{
			Year:  dobYear,
			Month: dobMonth,
			Day:   dobDay,
		},
	}
	referralDetails.Status.GDStatus = "referred"
	referralDetails.Status.SPStatus = "referred"
	const _24K = 256 << 20
	var documentFiles *multipart.Form
	if err = c.Request.ParseMultipartForm(_24K); err == nil {
		documentFiles = c.Request.MultipartForm
	}
	go processReferral(referralDetails, gproject, true, documentFiles)
	userID, err := getUserDetailsAnonymous(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
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
	idAuth.DeleteAnonymousUser(ctx, userID)
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "referral created successfully.",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

func processReferral(referralDetails contracts.ReferralDetails, gproject string, isQR bool, documentFiles *multipart.Form) (*contracts.DSReferral, *contracts.ReferralComments) {
	storageC := storage.NewStorageHandler()
	ctx := context.Background()
	err := storageC.InitializeStorageClient(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
	if err != nil {
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	currentRefUUID, _ := uuid.NewUUID()
	uniqueRefID := currentRefUUID.String()
	docsMedia := make([]contracts.Media, 0)
	docIDNames := make([]string, 0)

	// Stage 2 Upload files from
	// parse request

	foundImage := false
	if documentFiles != nil {
		for _, fheaders := range documentFiles.File {
			for _, hdr := range fheaders {
				// open uploaded
				var infile multipart.File
				if infile, err = hdr.Open(); err != nil {

					log.Errorf("Failed to created referral: %v", err.Error())
					return nil, nil

				}
				fileName := hdr.Filename

				reader, err := storageC.DownloadSingleFile(ctx, uniqueRefID, constants.SD_REFERRAL_BUCKET, fileName)
				if err == nil && reader != nil {
					timeNow := time.Now()
					stripFile := strings.Split(fileName, ".")
					name := stripFile[0]
					name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
					fileName = name + "." + stripFile[len(stripFile)-1]
				}
				bucketPath := uniqueRefID + "/" + fileName
				buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
				if err != nil {
					log.Errorf("Failed to created referral: %v", err.Error())
					return nil, nil
				}
				imageBuffer := bytes.NewBuffer(nil)
				if _, err := io.Copy(imageBuffer, infile); err != nil {

					log.Errorf("Failed to created referral: %v", err.Error())
					return nil, nil

				}
				currentBytes := imageBuffer.Bytes()
				if isQR {
					//client.SetImageFromBytes(currentBytes)
				}
				foundImage = true
				io.Copy(buckerW, bytes.NewReader(currentBytes))
				buckerW.Close()
				extentionFile := strings.Split(fileName, ".")
				ext := extentionFile[len(extentionFile)-1]
				var docMedia contracts.Media
				docMedia.Name = fileName
				docMedia.Image = ""
				switch strings.ToLower(ext) {
				case "jpg", "jpeg":
					img, err := jpeg.Decode(bytes.NewReader(currentBytes))
					if err == nil {
						resized := imaging.Resize(img, 200, 0, imaging.Lanczos)

						buf := bytes.NewBuffer(nil)
						err := jpeg.Encode(buf, resized, nil)
						if err == nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
							log.Info(docMedia.Image)
						}
					}
				case "png":
					img, err := png.Decode(bytes.NewReader(currentBytes))
					if err == nil {
						resized := imaging.Resize(img, 200, 0, imaging.Lanczos)
						buf := bytes.NewBuffer(nil)
						err := jpeg.Encode(buf, resized, nil)
						if err == nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				}
				docsMedia = append(docsMedia, docMedia)
				docIDNames = append(docIDNames, fileName)
			}
		}
		err = storageC.ZipFile(ctx, uniqueRefID, constants.SD_REFERRAL_BUCKET)
		if err != nil {
			log.Errorf("Failed to created referral: %v", err.Error())
			return nil, nil
		}
	}
	ocrText := ""
	if foundImage && isQR {
		//ocrText, _ = client.Text()
	}
	ocrText = "Please refer to attached documents for more details."
	var dsReferral contracts.DSReferral
	dsReferral.PatientDOBYear = referralDetails.Patient.Dob.Year
	dsReferral.PatientDOBMonth = referralDetails.Patient.Dob.Month
	dsReferral.PatientDOBDay = referralDetails.Patient.Dob.Day
	if ocrText != "" {
		startIndex := strings.Index(ocrText, "Reason")
		endIndex := strings.Index(ocrText, "Faster")

		if startIndex >= 0 && endIndex-1 > 0 {
			ocrText = ocrText[startIndex : endIndex-1]
		} else if startIndex >= 0 {
			ocrText = ocrText[startIndex:]
		}
		dsReferral.Reasons = []string{ocrText}
	}
	dsReferral.IsSummary = referralDetails.IsSummary
	if referralDetails.Patient.Phone != "" {
		currentPhone := referralDetails.Patient.Phone
		pnum, _ := phonenumbers.Parse(currentPhone, "US")
		countryCode := "+1"
		if pnum.CountryCode != nil {
			countryCode = "+" + strconv.Itoa(int(*pnum.CountryCode))
		}
		dsReferral.PatientPhone = countryCode + strconv.Itoa(int(*pnum.NationalNumber))
	}
	dsReferral.IsQR = isQR
	dsReferral.Documents = docIDNames
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
			log.Errorf("Failed to created referral: %v", err.Error())
			return nil, nil
		}
		dsReferral.FromPlaceID = fromClinic.PlaceID
		dsReferral.FromClinicName = fromClinic.Name
		dsReferral.FromClinicAddress = fromClinic.Address
		dsReferral.FromEmail = fromClinic.EmailAddress
		dsReferral.FromClinicPhone = fromClinic.PhoneNumber
		zone, err := tz.GetZone(tz.Point{
			Lon: fromClinic.Location.Long, Lat: fromClinic.Location.Lat,
		})
		location, _ := time.LoadLocation(zone[0])
		dsReferral.CreatedOn = time.Now().In(location)
		dsReferral.ModifiedOn = time.Now().In(location)
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
			location, _ := time.LoadLocation(zone[0])
			dsReferral.CreatedOn = time.Now().In(location)
			dsReferral.ModifiedOn = time.Now().In(location)
			if isQR {
				var commentReasons contracts.Comment
				currentID, _ := uuid.NewUUID()
				commentReasons.MessageID = currentID.String()
				commentReasons.Channel = "c2c"
				commentReasons.Text = "New QR based referral is created."
				location, _ := time.LoadLocation(zone[0])
				commentReasons.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
				commentReasons.UserID = fromClinic.EmailAddress
				updatedComm = append(updatedComm, commentReasons)
				currentID, _ = uuid.NewUUID()
				commentReasons.MessageID = currentID.String()
				commentReasons.Channel = "c2c"
				commentReasons.Text = "QR snapshot attached."
				//commentReasons.Files = docIDNames
				commentReasons.Media = docsMedia
				commentReasons.Files = docIDNames
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
			}
		} else {
			mapClient := gmaps.NewMapsHandler()
			err = mapClient.InitializeGoogleMapsAPIClient(ctx, gproject)
			if err != nil {
				log.Errorf("Failed to created referral: %v", err.Error())
				return nil, nil
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
			log.Errorf("Failed to created referral: %v", err.Error())
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
				log.Errorf("Failed to created referral: %v", err.Error())
				return nil, nil
			}
			details, err := mapClient.FindPlaceFromID(referralDetails.ToPlaceID)
			if err != nil {
				log.Errorf("Failed to created referral: %v", err.Error())
				return nil, nil
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
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	var returnComments contracts.ReferralComments
	returnComments.Comments = updatedComm
	var refComments contracts.ReferralComments
	refComments.Comments = append(refComments.Comments, returnComments.Comments...)
	_, err = ProcessComments(ctx, gproject, dsReferral.ReferralID, refComments)
	if err != nil {
		log.Errorf("Failed to created referral: %v", err.Error())
		return nil, nil
	}
	//err = clinicDB.AddPatientInformation(ctx, referralDetails.Patient)
	//if err != nil {
	//	log.Errorf("Failed to create patient information: %v", err.Error())
	//	return nil, nil
	//}
	return &dsReferral, &returnComments
}
