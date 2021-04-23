package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	//"github.com/otiai10/gosseract/v2"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/gsheets"
	"github.com/superdentist/superdentist-backend/lib/identity"
	"github.com/superdentist/superdentist-backend/lib/sms"
	"github.com/superdentist/superdentist-backend/lib/storage"
	"go.opencensus.io/trace"
	"gopkg.in/ugjka/go-tz.v2/tz"
)

// RegisterPatientInformation ....
func RegisterPatientInformation(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
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
	gproject := googleprojectlib.GetGoogleProjectID()
	userID, err := getUserDetailsAnonymous(ctx, c.Request)
	providerID, err := getProviderID(ctx, c.Request)
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
	if providerID != "" && strings.Contains(providerID, "anonymous") {
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
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "patient registration successful",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetAllPatientsForClinic ....
func GetAllPatientsForClinic(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	gproject := googleprojectlib.GetGoogleProjectID()
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	// here is we have referral id
	addressID := c.Param("addressId")
	patientDB := datastoredb.NewPatientHandler()
	pageSize, err := strconv.Atoi(c.Query("pageSize"))
	if err != nil {
		pageSize = 0
	}
	cursor := c.Query("cursor")
	err = patientDB.InitializeDataBase(ctx, gproject)
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
	if pageSize == 0 {
		patients := patientDB.GetPatientByAddressID(ctx, addressID)
		c.JSON(http.StatusOK, gin.H{
			constants.RESPONSE_JSON_DATA:   patients,
			constants.RESPONSDE_JSON_ERROR: nil,
		})
	} else {
		patients, cursor := patientDB.GetPatientByAddressIDPaginate(ctx, addressID, pageSize, cursor)
		var patientsList contracts.PatientList
		patientsList.Patients = patients
		patientsList.Cursor = cursor
		c.JSON(http.StatusOK, gin.H{
			constants.RESPONSE_JSON_DATA:   patients,
			constants.RESPONSDE_JSON_ERROR: nil,
		})
	}
}

// AddPatientNotes ....
func AddPatientNotes(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	pID := c.Param("patientId")

	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	var patientNotes contracts.Notes
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	patientNotes.Details = string(bodyBytes)
	gproject := googleprojectlib.GetGoogleProjectID()

	patientDB := datastoredb.NewPatientHandler()
	err = patientDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	patientNotes.PatientID = pID
	err = patientDB.AddPatientNotes(ctx, patientNotes)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "patient registration successful",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// UpdatePatientStatus ....
func UpdatePatientStatus(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	pID := c.Param("patientId")
	ctx, span := trace.StartSpan(ctx, "Updating Patient Status")
	defer span.End()
	gproject := googleprojectlib.GetGoogleProjectID()
	var pStatus contracts.PatientStatus
	if err := c.ShouldBindWith(&pStatus, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	patientDB := datastoredb.NewPatientHandler()
	err := patientDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	err = patientDB.UpdatePatientStatus(ctx, pID, pStatus)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "patient status update successful",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetPatientNotes ....
func GetPatientNotes(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	pID := c.Param("patientId")

	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	gproject := googleprojectlib.GetGoogleProjectID()

	patientDB := datastoredb.NewPatientHandler()
	err := patientDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	notes, err := patientDB.GetAddPatientNotes(ctx, pID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   notes.Details,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// UploadPatientDocuments ....
func UploadPatientDocuments(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	// here is we have referral id
	pID := c.Param("patientId")

	const _24K = 256 << 20
	var documentFiles *multipart.Form
	if err := c.Request.ParseMultipartForm(_24K); err == nil {
		documentFiles = c.Request.MultipartForm
	}
	err := uploadPatientDocs(ctx, pID, documentFiles)
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
		constants.RESPONSE_JSON_DATA:   "patient registration successful",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// ProcessPatientSpreadSheet ....
func ProcessPatientSpreadSheet(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	// here is we have referral id
	gproject := googleprojectlib.GetGoogleProjectID()

	const _24K = 256 << 20
	var documentFiles *multipart.Form
	if err := c.Request.ParseMultipartForm(_24K); err == nil {
		documentFiles = c.Request.MultipartForm
	}
	if documentFiles == nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("no sheets provided to backend to process"),
			},
		)
		return
	}

	err := processSpreadSheet(ctx, gproject, documentFiles)
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
		case "zipCode":
			patientDetails.ZipCode = fieldValue[0]
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
		case "addressId":
			addId := fieldValue[0]
			patientDetails.SameDay = true
			patientDetails.AddressID = addId
		}
	}
	gproject := googleprojectlib.GetGoogleProjectID()
	ctx := context.Background()
	var dsReferral *contracts.DSReferral
	clinicDB := datastoredb.NewClinicMetaHandler()
	err := clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to initialize clinics: %v", err.Error())
		return err
	}
	if patientDetails.AddressID != "" {

		currentClinic, err := clinicDB.GetSingleClinic(ctx, patientDetails.AddressID)
		if err != nil {
			log.Errorf("Failed to get clinic: %v", err.Error())
			return err
		}
		patientDetails.ClinicName = currentClinic.Name
		zone, err := tz.GetZone(tz.Point{
			Lon: currentClinic.Location.Long, Lat: currentClinic.Location.Lat,
		})
		location, _ := time.LoadLocation(zone[0])
		patientDetails.DueDate = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
		patientDetails.CreatedOn = patientDetails.DueDate
		patientDetails.CreationDate = time.Now().In(location).Format("2006-01-02 15:04:05")
	}
	if refID != "" {
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
			currentClinic, err := clinicDB.GetSingleClinic(ctx, patientDetails.GD)
			if err != nil {
				log.Errorf("Failed to get clinic: %v", err.Error())
				return err
			}
			zone, err := tz.GetZone(tz.Point{
				Lon: currentClinic.Location.Long, Lat: currentClinic.Location.Lat,
			})
			location, _ := time.LoadLocation(zone[0])
			patientDetails.CreatedOn = patientDetails.DueDate
			patientDetails.CreationDate = time.Now().In(location).Format("2006-01-02 15:04:05")
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
	}

	storageC := storage.NewStorageHandler()
	err = storageC.InitializeStorageClient(ctx, gproject)
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
	patientID, _ := uuid.NewUUID()
	pIDString := patientID.String()
	patientDetails.PatientID = pIDString
	key, err := patientDB.AddPatientInformation(ctx, patientDetails, pIDString)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	patientFolder := key
	googleSheet := gsheets.NewSheetsHandler()
	googleSheet.InitializeSheetsClient(ctx, gproject)
	err = googleSheet.WritePatientoGSheet(patientDetails, "12A93KjDeO4eVEUYwunzLZxKx4HkqjI19HrCDjhp85Q8")
	if err != nil {
		log.Errorf("Sheet write error: %v", err.Error())
	}
	if documentFiles != nil && documentFiles.File != nil && len(documentFiles.File) > 0 {
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
					fileName = name + "." + stripFile[len(stripFile)-1]
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
	if dsReferral != nil && patientDetails.AddressID == "" && dsReferral.CommunicationText != "" && dsReferral.CommunicationPhone != "" {
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

func uploadPatientDocs(ctx context.Context, patientFolder string, documentFiles *multipart.Form) error {
	gproject := googleprojectlib.GetGoogleProjectID()
	storageC := storage.NewStorageHandler()
	err := storageC.InitializeStorageClient(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
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
					fileName = name + "." + stripFile[len(stripFile)-1]
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
	return nil
}

func processSpreadSheet(ctx context.Context, gproject string, documentFiles *multipart.Form) error {
	var dueDate int64
	var addressID string
	for fieldName, fieldValue := range documentFiles.Value {
		switch fieldName {
		case "dueDate":
			dueDate, _ = strconv.ParseInt(fieldValue[0], 10, 64)
		case "addressId":
			addressID = fieldValue[0]
		}
	}
	var err error
	log.Infof("Due Date: %v", dueDate)
	if addressID == "" || dueDate <= 0 {
		return fmt.Errorf("missing required information: check clinic address id (addressId) or due date (dueDate)")
	}
	for _, xlFileValue := range documentFiles.File {

		for _, hdr := range xlFileValue {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); err != nil {

				log.Errorf("Failed to created patient information: %v", err.Error())
				return err

			}
			fileName := hdr.Filename
			splitNameFile := strings.Split(fileName, ".")
			foundSheet := false
			switch splitNameFile[len(splitNameFile)-1] {
			case "csv", "xlsx":
				foundSheet = true
			}
			if !foundSheet {
				continue
			}
			xlBytes := bytes.NewBuffer(nil)
			if _, err := io.Copy(xlBytes, infile); err != nil {

				log.Errorf("Failed to create patient information via excel file: %v", err.Error())
				return err

			}
			csvReader := csv.NewReader(xlBytes)
			err = processXlDocument(ctx, gproject, addressID, dueDate, csvReader)
			// read the file
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processXlDocument(ctx context.Context, gproject string, addressID string, dueDate int64, xlFile *csv.Reader) error {
	columnMap := make(map[int]string)
	patientDB := datastoredb.NewPatientHandler()
	err := patientDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	// Iterate through the records
	counter := 0
	for {
		// Read each record from csv
		record, err := xlFile.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("Failed to created patient information: %v", err.Error())
			return err
		}
		var patientInfo contracts.Patient
		for i, text := range record {
			if counter == 0 {
				columnMap[i] = strings.ToLower(text)
			} else {
				patientInfo.DueDate = dueDate
				switch columnMap[i] {
				case "time":
					patientInfo.AppointmentTime = text
				case "name":
					splitName := strings.Split(text, ",")
					patientInfo.LastName = splitName[0]
					patientInfo.FirstName = splitName[1]
				case "dob":
					splitDOB := strings.Split(text, "/")
					patientInfo.Dob = contracts.DOB{
						Year:  splitDOB[2],
						Day:   splitDOB[1],
						Month: splitDOB[0],
					}
				case "patient zip":
					patientInfo.ZipCode = text
				case "zip":
					patientInfo.ZipCode = text
				case "zipcode":
					patientInfo.ZipCode = text
				case "subscriber":
					var subscriber contracts.Subscriber
					splitName := strings.Split(text, ",")
					subscriber.LastName = splitName[0]
					subscriber.FirstName = splitName[1]
					patientInfo.DentalInsurance = make([]contracts.PatientDentalInsurance, 0)
					var dInsurance contracts.PatientDentalInsurance
					dInsurance.Subscriber = subscriber
					patientInfo.DentalInsurance = append(patientInfo.DentalInsurance, dInsurance)
				case "subscriber dob":
					if len(patientInfo.DentalInsurance) > 0 {
						splitDOB := strings.Split(text, "/")
						sdob := contracts.DOB{
							Year:  splitDOB[2],
							Day:   splitDOB[1],
							Month: splitDOB[0],
						}
						for i, insurance := range patientInfo.DentalInsurance {
							insurance.Subscriber.DOB = sdob
							patientInfo.DentalInsurance[i] = insurance
						}
					}
				case "insurance":
					for i, insurance := range patientInfo.DentalInsurance {
						insurance.Company = text
						patientInfo.DentalInsurance[i] = insurance
					}
				case "id":
					for i, insurance := range patientInfo.DentalInsurance {
						insurance.MemberID = text
						patientInfo.DentalInsurance[i] = insurance
					}
				}
			}
		}
		if counter > 0 && patientInfo.FirstName != "" {
			patientInfo.GD = addressID
			patientID, _ := uuid.NewUUID()
			pIDString := patientID.String()
			patientInfo.PatientID = pIDString
			patientDB.AddPatientInformation(ctx, patientInfo, pIDString)
		}
		counter++
	}
	return nil
}
