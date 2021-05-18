package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	agentID := c.Query("agentId")
	var startTimeStamp int64
	var endTimeStamp int64

	startTimeStamp = 0
	endTimeStamp = 0
	var err error
	var filters contracts.PatientFilters
	if startTime != "" {
		startTimeStamp, err = strconv.ParseInt(startTime, 10, 64)
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
		filters.StartTime = startTimeStamp
	}
	if endTime != "" {
		endTimeStamp, err = strconv.ParseInt(endTime, 10, 64)
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
		filters.EndTime = endTimeStamp

	}
	filters.AgentID = agentID
	filteringRequested := false
	if filters.StartTime > 0 || filters.AgentID != "" {
		filteringRequested = true
	}
	patientDB := datastoredb.NewPatientHandler()
	pageSize, err := strconv.Atoi(c.Query("pageSize"))
	if err != nil {
		pageSize = 0
	}
	cursor := c.Query("cursor")
	cursorPrev := cursor
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
		if filteringRequested {
			patients := patientDB.GetPatientByFilters(ctx, addressID, filters)
			c.JSON(http.StatusOK, gin.H{
				constants.RESPONSE_JSON_DATA:   patients,
				constants.RESPONSDE_JSON_ERROR: nil,
			})
		} else {
			patients := patientDB.GetPatientByAddressID(ctx, addressID)
			patientsList := patientDB.ReturnPatientsWithDMInsurancesArr(ctx, patients)
			c.JSON(http.StatusOK, gin.H{
				constants.RESPONSE_JSON_DATA:   patientsList,
				constants.RESPONSDE_JSON_ERROR: nil,
			})
		}
	} else {
		if filteringRequested {
			patients, cursor := patientDB.GetPatientByFiltersPaginate(ctx, addressID, filters, pageSize, cursor)
			var patientsList contracts.PatientList
			patientsList.Patients = patients
			patientsList.CursorPrev = cursorPrev
			patientsList.CursorNext = cursor
			c.JSON(http.StatusOK, gin.H{
				constants.RESPONSE_JSON_DATA:   patientsList,
				constants.RESPONSDE_JSON_ERROR: nil,
			})
		} else {
			patientStore, cursor := patientDB.GetPatientByAddressIDPaginate(ctx, addressID, pageSize, cursor)
			var patientsList contracts.PatientList
			patients := patientDB.ReturnPatientsWithDMInsurances(ctx, patientStore)
			patientsList.Patients = patients
			patientsList.CursorPrev = cursorPrev
			patientsList.CursorNext = cursor
			c.JSON(http.StatusOK, gin.H{
				constants.RESPONSE_JSON_DATA:   patientsList,
				constants.RESPONSDE_JSON_ERROR: nil,
			})
		}
	}
}

// GetAllPatientsForClinic ....
func GetSinglePatientForClinic(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	gproject := googleprojectlib.GetGoogleProjectID()
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	defer span.End()
	// here is we have referral id
	pID := c.Param("patientId")
	patientDB := datastoredb.NewPatientHandler()

	err := patientDB.InitializeDataBase(ctx, gproject)
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

	patients, _, _, err := patientDB.GetPatientByID(ctx, pID)
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
		constants.RESPONSE_JSON_DATA:   patients,
		constants.RESPONSDE_JSON_ERROR: nil,
	})

}

// AddAgentToPatient ....
func AddAgentToPatient(c *gin.Context) {
	// // Stage 1  Load the incoming request
	// log.Infof("Patient Stuff")
	// gproject := googleprojectlib.GetGoogleProjectID()
	// ctx := c.Request.Context()
	// ctx, span := trace.StartSpan(ctx, "Register incoming request for clinic")
	// defer span.End()
	// // here is we have referral id
	// pID := c.Param("patientId")
	// patientDB := datastoredb.NewPatientHandler()
	// agentID := c.Param("agentId")

	// err := patientDB.InitializeDataBase(ctx, gproject)
	// if err != nil {
	// 	c.AbortWithStatusJSON(
	// 		http.StatusInternalServerError,
	// 		gin.H{
	// 			constants.RESPONSE_JSON_DATA:   nil,
	// 			constants.RESPONSDE_JSON_ERROR: err.Error(),
	// 		},
	// 	)
	// 	return
	// }

	// patients, _, err := patientDB.GetPatientByID(ctx, pID)
	// if err != nil {
	// 	c.AbortWithStatusJSON(
	// 		http.StatusInternalServerError,
	// 		gin.H{
	// 			constants.RESPONSE_JSON_DATA:   nil,
	// 			constants.RESPONSDE_JSON_ERROR: err.Error(),
	// 		},
	// 	)
	// 	return
	// }
	// patients.AgentID = agentID
	// _, err = patientDB.AddPatientInformation(ctx, *patients, patients.PatientID)
	// if err != nil {
	// 	c.AbortWithStatusJSON(
	// 		http.StatusBadRequest,
	// 		gin.H{
	// 			constants.RESPONSE_JSON_DATA:   nil,
	// 			constants.RESPONSDE_JSON_ERROR: fmt.Errorf("failed to add agent for patient"),
	// 		},
	// 	)
	// 	return
	// }
	// c.JSON(http.StatusOK, gin.H{
	// 	constants.RESPONSE_JSON_DATA:   "agent registered for patient",
	// 	constants.RESPONSDE_JSON_ERROR: nil,
	// })

}

// AddPatientNotes ....
func AddPatientNotes(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	pID := c.Param("patientId")
	notesType := c.Query("notesType")
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
	patientNotes.Type = notesType
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
func UpdateInsurnaceStatus(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	insuranceID := c.Param("insuranceId")
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
	err = patientDB.UpdateInsuranceStatus(ctx, insuranceID, pStatus)
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

// AddInsuranceAgent ....
func AddInsuranceAgent(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	insuranceID := c.Param("insuranceId")
	agentID := c.Param("agentId")
	ctx, span := trace.StartSpan(ctx, "Updating Patient Agent")
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
	err = patientDB.AddAgentToInsurance(ctx, insuranceID, agentID)
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
		constants.RESPONSE_JSON_DATA:   "agent added successfully",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// AddInsuranceAgents ....
func AddInsuranceAgents(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Patient Stuff")
	ctx := c.Request.Context()
	// here is we have referral id
	ctx, span := trace.StartSpan(ctx, "Updating Patient Agent")
	agentInsuraneMap := make([]contracts.AgentInsuranceMap, 0)
	defer span.End()
	if err := c.ShouldBindWith(&agentInsuraneMap, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
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
	for _, aiMap := range agentInsuraneMap {
		insuranceID := aiMap.InsuranceID
		agentID := aiMap.AgentID
		err = patientDB.AddAgentToInsurance(ctx, insuranceID, agentID)
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
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "agents added successfully",
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
	notesType := c.Query("notesType")
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
	notes, err := patientDB.GetAddPatientNotes(ctx, pID+notesType)
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

func registerPatientInDB(documentFiles *multipart.Form) error {
	var patientDetails contracts.PatientStore
	dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
	medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)

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
			dInsuranceIDs := make([]string, 0)
			dentalInsurance1 := make([]contracts.PatientDentalInsurance, 0)
			err := json.Unmarshal([]byte(fieldValue[0]), &dentalInsurance1)
			if len(dentalInsurance1) > 0 && err == nil {
				for _, insurance := range dentalInsurance1 {
					insurance.MemberID = strings.TrimSpace(insurance.MemberID)
					insurance.ID = insurance.MemberID
					if insurance.ID != "" && insurance.ID != "0" {
						insurance.Status = contracts.PatientStatus{Label: "Pending", Value: "pending"}
						dentalInsurance = append(dentalInsurance, insurance)
						dInsuranceIDs = append(dInsuranceIDs, insurance.ID)
					}
				}
				patientDetails.DentalInsuraceID = dInsuranceIDs

			}
		case "medicalInsurance":
			mInsuranceIDs := make([]string, 0)
			medicalInsurance1 := make([]contracts.PatientMedicalInsurance, 0)

			err := json.Unmarshal([]byte(fieldValue[0]), &medicalInsurance1)
			if len(medicalInsurance1) > 0 && err == nil {
				for _, insurance := range medicalInsurance1 {
					insurance.GroupNumber = strings.TrimSpace(insurance.GroupNumber)
					insurance.SSN = strings.TrimSpace(insurance.SSN)
					insurance.MemberID = strings.TrimSpace(insurance.MemberID)
					insurance.ID = insurance.MemberID + insurance.GroupNumber + insurance.SSN
					if insurance.ID != "" && insurance.ID != "0" {
						insurance.Status = contracts.PatientStatus{Label: "Pending", Value: "pending"}
						medicalInsurance = append(medicalInsurance, insurance)
						mInsuranceIDs = append(mInsuranceIDs, insurance.ID)
					}
				}
				patientDetails.MedicalInsuranceID = mInsuranceIDs
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
	key, err := patientDB.AddPatientInformation(ctx, patientDetails, pIDString, dentalInsurance, medicalInsurance)
	if err != nil {
		log.Errorf("Failed to created patient information: %v", err.Error())
		return err
	}
	patientFolder := key
	var patientStore contracts.Patient
	patientStore.AddressID = patientDetails.AddressID
	patientStore.ClinicName = patientDetails.ClinicName
	patientStore.FirstName = patientDetails.FirstName
	patientStore.LastName = patientDetails.LastName
	patientStore.Dob = patientDetails.Dob
	patientStore.Email = patientDetails.Email
	patientStore.GD = patientDetails.GD
	patientStore.GDName = patientDetails.GDName
	patientStore.SP = patientDetails.SP
	patientStore.SPName = patientDetails.SPName
	patientStore.SSN = patientDetails.SSN
	patientStore.SameDay = patientDetails.SameDay
	patientStore.Phone = patientDetails.Phone
	patientStore.SSN = patientDetails.SSN
	patientStore.ZipCode = patientDetails.ZipCode
	patientStore.Status = patientDetails.Status
	patientStore.DueDate = patientDetails.DueDate
	patientStore.AppointmentTime = patientDetails.AppointmentTime
	patientStore.CreatedOn = patientDetails.CreatedOn
	patientStore.CreationDate = patientDetails.CreationDate
	patientStore.PatientID = patientDetails.PatientID
	patientStore.DentalInsurance = dentalInsurance
	patientStore.MedicalInsurance = medicalInsurance
	googleSheet := gsheets.NewSheetsHandler()
	googleSheet.InitializeSheetsClient(ctx, gproject)
	err = googleSheet.WritePatientoGSheet(patientStore, "12A93KjDeO4eVEUYwunzLZxKx4HkqjI19HrCDjhp85Q8")
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
