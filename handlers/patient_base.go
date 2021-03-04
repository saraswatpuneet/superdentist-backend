package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	//"github.com/otiai10/gosseract/v2"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/sms"
	"github.com/superdentist/superdentist-backend/lib/storage"
	"go.opencensus.io/trace"
)

// RegisterPatientInformation ....
func RegisterPatientInformation(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Creating Referral")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	// here is we have referral id

	const _24K = 256 << 20
	var documentFiles *multipart.Form
	if err := c.Request.ParseMultipartForm(_24K); err == nil {
		documentFiles = c.Request.MultipartForm
	}
	go registerPatientInDB(documentFiles)
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "patient registration successful",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}
func registerPatientInDB(documentFiles *multipart.Form) error {
	var patientDetails contracts.Patient
	refID := ""
	for fieldName, fieldValue := range documentFiles.Value {
		switch fieldName {
		case "firstName":
			patientDetails.FirstName = fieldValue[0]
		case "lastName":
			patientDetails.LastName = fieldValue[0]
		case "phone":
			patientDetails.Phone = fieldValue[0]
		case "email":
			patientDetails.Email = fieldValue[0]
		case "ssn":
			patientDetails.SSN = fieldValue[0]
		case "dob":
			var dobStruct contracts.DOB
			err := json.Unmarshal([]byte(fieldValue[0]), &dobStruct)
			if err == nil {
				patientDetails.Dob = dobStruct
			}
		case "dentalInsurance":
			dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
			err := json.Unmarshal([]byte(fieldValue[0]), &dentalInsurance)
			if len(dentalInsurance) > 0 && err == nil {
				patientDetails.DentalInsurance = dentalInsurance
			}
		case "medicalInsurance":
			medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
			err := json.Unmarshal([]byte(fieldValue[0]), &medicalInsurance)
			if len(medicalInsurance) > 0 && err == nil {
				patientDetails.MedicalInsurance = medicalInsurance
			}
		case "referralId":
			refID = fieldValue[0]
		}
	}
	gproject := googleprojectlib.GetGoogleProjectID()
	ctx := context.Background()
	var dsReferral *contracts.DSReferral
	if refID != "" {
		refID = strings.Replace(refID, "_", "-", -1)
		dsRefC := datastoredb.NewReferralHandler()
		err := dsRefC.InitializeDataBase(ctx, gproject)
		if err != nil {
			log.Errorf("Failed to created patient information: %v", err.Error())
			return err
		}
		dsReferral, err = dsRefC.GetReferral(ctx, refID)
		if err != nil {
			log.Errorf("Failed to created patient information: %v", err.Error())
			return err
		}
		if dsReferral.FromAddressID != "" {
			patientDetails.GD = dsReferral.FromAddressID
		} else {
			patientDetails.GD = dsReferral.FromPlaceID
		}
		if dsReferral.ToAddressID != "" {
			patientDetails.SP = dsReferral.ToAddressID
		} else {
			patientDetails.SP = dsReferral.ToPlaceID
		}
		patientDetails.GDName = dsReferral.FromClinicName
		patientDetails.SPName = dsReferral.ToClinicName
		patientDetails.ReferralID = refID
	} else {
		return fmt.Errorf("bad referral id provided")
	}

	storageC := storage.NewStorageHandler()
	err := storageC.InitializeStorageClient(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	patientDB := datastoredb.NewPatientHandler()
	err = patientDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	key, err := patientDB.AddPatientInformation(ctx, patientDetails)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	patientFolder := key
	if documentFiles != nil {
		for _, fheaders := range documentFiles.File {
			for _, hdr := range fheaders {
				// open uploaded
				var infile multipart.File
				if infile, err = hdr.Open(); err != nil {

					log.Errorf("Failed to created patient information: %v", err.Error())
					return err

				}
				fileName := hdr.Filename

				reader, err := storageC.DownloadSingleFile(ctx, patientFolder, constants.SD_PATIENT_BUCKET, fileName)
				if err == nil && reader != nil {
					timeNow := time.Now()
					stripFile := strings.Split(fileName, ".")
					name := stripFile[0]
					name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
					fileName = name + "." + stripFile[1]
				}
				bucketPath := patientFolder + "/" + fileName
				buckerW, err := storageC.UploadToGCSPatient(ctx, bucketPath)
				if err != nil {
					log.Errorf("Failed to created patient information: %v", err.Error())
					return err
				}
				imageBuffer := bytes.NewBuffer(nil)
				if _, err := io.Copy(imageBuffer, infile); err != nil {

					log.Errorf("Failed to created patient information: %v", err.Error())
					return err

				}
				currentBytes := imageBuffer.Bytes()
				io.Copy(buckerW, bytes.NewReader(currentBytes))
				buckerW.Close()
			}
		}
		err = storageC.ZipFile(ctx, patientFolder, constants.SD_PATIENT_BUCKET)
		if err != nil {
			log.Errorf("Failed to created patient information: %v", err.Error())
			return err
		}
	}
	if dsReferral.CommunicationText != "" && dsReferral.CommunicationPhone != "" {
		clientSMS := sms.NewSMSClient()
		err = clientSMS.InitializeSMSClient()
		if err != nil {
			log.Errorf("Failed to send SMS: %v", err.Error())
			return err
		}
		message2 := dsReferral.CommunicationText
		if message2 != "" {
			err = clientSMS.SendSMS(dsReferral.CommunicationPhone, dsReferral.PatientPhone, message2)
		}
	}
	return nil
}
