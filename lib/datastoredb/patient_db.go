package datastoredb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"cloud.google.com/go/datastore"
)

// DSPatient ...
type DSPatient struct {
	projectID string
	client    *datastore.Client
}

//NewPatientHandler return new database action
func NewPatientHandler() *DSPatient {
	return &DSPatient{projectID: "", client: nil}
}

// InitializeDataBase ....
func (db *DSPatient) InitializeDataBase(ctx context.Context, projectID string) error {
	serviceAccountSD := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountSD == "" {
		return fmt.Errorf("Failed to get right credentials for superdentist backend")
	}
	targetScopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	}
	currentCreds, _, err := helpers.ReadCredentialsFile(ctx, serviceAccountSD, targetScopes)
	dsClient, err := datastore.NewClient(context.Background(), projectID, option.WithCredentials(currentCreds))
	if err != nil {
		return err
	}
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := dsClient.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	db.client = dsClient
	db.projectID = projectID
	return nil
}

// AddPatientInformation ....
func (db DSPatient) AddPatientInformation(ctx context.Context, patient contracts.PatientStore, pIDString string, dI []contracts.PatientDentalInsurance, mI []contracts.PatientMedicalInsurance) (string, error) {

	pKey := datastore.NameKey("PatientIndexed", pIDString, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	patient.PatientID = pIDString
	_, err := db.client.Put(ctx, pKey, &patient)
	if err != nil {
		return "", err
	}
	for _, insurance := range dI {
		pKey := datastore.NameKey("DentalInsuranceIndexed", insurance.ID, nil)
		insurance.PatientID = pIDString
		insurance.DueDate = patient.DueDate

		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &insurance)
		if err != nil {
			return "", err
		}
	}
	for _, insurance := range mI {
		pKey := datastore.NameKey("MedicalInsuranceIndexed", insurance.ID, nil)
		insurance.PatientID = pIDString
		insurance.DueDate = patient.DueDate
		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &insurance)
		if err != nil {
			return "", err
		}
	}
	return pIDString, nil
}

// AddPatientInformation ....
func (db DSPatient) AddPatientInformationStatus(ctx context.Context, patient contracts.PatientStore, pIDString string) (string, error) {

	pKey := datastore.NameKey("PatientIndexed", pIDString, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	patient.PatientID = pIDString
	_, err := db.client.Put(ctx, pKey, &patient)
	if err != nil {
		return "", err
	}
	return pIDString, nil
}

// GetPatientInformation ....
func (db DSPatient) GetAddPatientNotes(ctx context.Context, pIDString string) (contracts.Notes, error) {
	var regularInterface contracts.Notes

	pKey := datastore.NameKey("PatientNotes", pIDString, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	err := db.client.Get(ctx, pKey, &regularInterface)
	if err != nil {
		return regularInterface, err
	}
	return regularInterface, nil
}

// AddPatientNotes ....
func (db DSPatient) AddPatientNotes(ctx context.Context, notes contracts.Notes) error {
	pKey := datastore.NameKey("PatientNotes", notes.PatientID+notes.Type, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	_, err := db.client.Put(ctx, pKey, &notes)
	if err != nil {
		return err
	}
	return nil
}

// GetPatientByFilters ...
func (db DSPatient) GetPatientByFilters(ctx context.Context, addressID string, filters contracts.PatientFilters) []contracts.Patient {
	patientsMap := make(map[string]contracts.Patient, 0)

	if filters.Companies != nil && len(filters.Companies) > 0 {
		for company := range filters.Companies {
			dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
			medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
			qP := datastore.NewQuery("DentalInsuranceIndexed")
			qP = qP.Filter("AddressID=", addressID)
			if filters.StartTime > 0 && filters.EndTime > 0 {
				qP = qP.Filter("DueDate >=", filters.StartTime)
				qP = qP.Filter("DueDate <=", filters.EndTime)
			}
			if filters.AgentID != "" {
				qP = qP.Filter("AgentID =", filters.AgentID)
			}
			qP.Filter("Company =", company)
			if global.Options.DSName != "" {
				qP = qP.Namespace(global.Options.DSName)
			}
			db.client.GetAll(ctx, qP, &dentalInsurance)

			qP = datastore.NewQuery("MedicalInsuranceIndexed")
			qP = qP.Filter("AddressID=", addressID)
			if filters.StartTime > 0 && filters.EndTime > 0 {
				qP = qP.Filter("DueDate >=", filters.StartTime)
				qP = qP.Filter("DueDate <=", filters.EndTime)
			}
			if filters.AgentID != "" {
				qP = qP.Filter("AgentID =", filters.AgentID)
			}
			if global.Options.DSName != "" {
				qP = qP.Namespace(global.Options.DSName)
			}
			qP.Filter("Company =", company)

			db.client.GetAll(ctx, qP, &medicalInsurance)
			if len(medicalInsurance) > 0 {
				for _, insurance := range medicalInsurance {
					if patient, ok := patientsMap[insurance.PatientID]; ok {
						patient.MedicalInsurance = append(patient.MedicalInsurance, insurance)
						patientsMap[patient.PatientID] = patient
					} else {
						patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
						if err == nil {
							patientOne.MedicalInsurance = append(patientOne.MedicalInsurance, insurance)
							patientsMap[patientOne.PatientID] = *patientOne
						}
					}
				}
			}
			if len(dentalInsurance) > 0 {
				for _, insurance := range dentalInsurance {
					if patient, ok := patientsMap[insurance.PatientID]; ok {
						patient.DentalInsurance = append(patient.DentalInsurance, insurance)
						patientsMap[patient.PatientID] = patient
					} else {
						patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
						if err == nil {
							patientOne.DentalInsurance = append(patientOne.DentalInsurance, insurance)
							patientsMap[patientOne.PatientID] = *patientOne
						}
					}
				}
			}

		}
	} else {
		dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
		medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
		qP := datastore.NewQuery("DentalInsuranceIndexed")
		qP = qP.Filter("AddressID=", addressID)
		if filters.StartTime > 0 && filters.EndTime > 0 {
			qP = qP.Filter("DueDate >=", filters.StartTime)
			qP = qP.Filter("DueDate <=", filters.EndTime)
		}
		if filters.AgentID != "" {
			qP = qP.Filter("AgentID =", filters.AgentID)
		}
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		db.client.GetAll(ctx, qP, &dentalInsurance)

		qP = datastore.NewQuery("MedicalInsuranceIndexed")
		qP = qP.Filter("AddressID=", addressID)
		if filters.StartTime > 0 && filters.EndTime > 0 {
			qP = qP.Filter("DueDate >=", filters.StartTime)
			qP = qP.Filter("DueDate <=", filters.EndTime)
		}
		if filters.AgentID != "" {
			qP = qP.Filter("AgentID =", filters.AgentID)
		}
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		db.client.GetAll(ctx, qP, &medicalInsurance)
		if len(medicalInsurance) > 0 {
			for _, insurance := range medicalInsurance {
				if patient, ok := patientsMap[insurance.PatientID]; ok {
					patient.MedicalInsurance = append(patient.MedicalInsurance, insurance)
					patientsMap[patient.PatientID] = patient
				} else {
					patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
					if err == nil {
						patientOne.MedicalInsurance = append(patientOne.MedicalInsurance, insurance)
						patientsMap[patientOne.PatientID] = *patientOne
					}
				}
			}
		}
		if len(dentalInsurance) > 0 {
			for _, insurance := range dentalInsurance {
				if patient, ok := patientsMap[insurance.PatientID]; ok {
					patient.DentalInsurance = append(patient.DentalInsurance, insurance)
					patientsMap[patient.PatientID] = patient
				} else {
					patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
					if err == nil {
						patientOne.DentalInsurance = append(patientOne.DentalInsurance, insurance)
						patientsMap[patientOne.PatientID] = *patientOne
					}
				}
			}
		}
	}
	patients := make([]contracts.Patient, 0)
	for _, patient := range patientsMap {
		patients = append(patients, patient)
	}
	return patients
}

// GetPatientByFilters ...
func (db DSPatient) GetPatientByNames(ctx context.Context, addressID string, firstName string, lastName string) []contracts.Patient {
	patientsMap := make(map[string]contracts.PatientStore, 0)
	if firstName == lastName {
		qP1 := datastore.NewQuery("PatientIndexed")
		qP1 = qP1.Filter("AddressID=", addressID)
		qP1 = qP1.Filter("FirstName=", firstName)
		qP2 := datastore.NewQuery("PatientIndexed")
		qP2 = qP2.Filter("AddressID=", addressID)
		qP2 = qP2.Filter("LastName=", lastName)
		if global.Options.DSName != "" {
			qP1 = qP1.Namespace(global.Options.DSName)
			qP2 = qP2.Namespace(global.Options.DSName)
		}
		patientsArr := make([]contracts.PatientStore, 0)
		_, err := db.client.GetAll(ctx, qP1, &patientsArr)
		if len(patientsArr) > 0 && err == nil {
			for _, pat := range patientsArr {
				if (pat.MedicalInsuranceID != nil && len(pat.MedicalInsuranceID) > 0) || (pat.DentalInsuraceID != nil && len(pat.DentalInsuraceID) > 0) {
					patientsMap[pat.PatientID] = pat

				}

			}
		}
		_, err = db.client.GetAll(ctx, qP2, &patientsArr)
		if len(patientsArr) > 0 && err == nil {
			for _, pat := range patientsArr {
				if (pat.MedicalInsuranceID != nil && len(pat.MedicalInsuranceID) > 0) || (pat.DentalInsuraceID != nil && len(pat.DentalInsuraceID) > 0) {
					patientsMap[pat.PatientID] = pat

				}

			}
		}

	} else {
		qP := datastore.NewQuery("PatientIndexed")
		qP = qP.Filter("AddressID=", addressID)
		qP = qP.Filter("FirstName=", firstName)
		qP = qP.Filter("LastName=", lastName)
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		patientsArr := make([]contracts.PatientStore, 0)
		db.client.GetAll(ctx, qP, &patientsArr)
		if len(patientsArr) > 0 {
			for _, pat := range patientsArr {
				if (pat.MedicalInsuranceID != nil && len(pat.MedicalInsuranceID) > 0) || (pat.DentalInsuraceID != nil && len(pat.DentalInsuraceID) > 0) {
					patientsMap[pat.PatientID] = pat

				}

			}
		}
	}
	patientsReturn := make([]contracts.Patient, 0)
	if len(patientsMap) > 0 {
		patientsReturn = db.ReturnPatientsWithDMInsurances(ctx, patientsMap)
	}

	return patientsReturn
}

// GetPatientByFiltersPaginate ...
func (db DSPatient) GetPatientByFiltersPaginate(ctx context.Context, addressID string, filters contracts.PatientFilters, pageSize int, cursor string) ([]contracts.Patient, string) {
	patientsMap := make(map[string]contracts.Patient, 0)
	mainCursor := ""
	if filters.Companies != nil && len(filters.Companies) > 0 {
		for _, company := range filters.Companies {
			dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
			medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
			qP := datastore.NewQuery("DentalInsuranceIndexed").Limit(pageSize)
			qP = qP.Filter("AddressID=", addressID)
			if cursor == "" {
				cursor = "cursor_" + "cursor_"
			}
			cursors := strings.Split(cursor, "cursor_")
			if filters.StartTime > 0 && filters.EndTime > 0 {
				qP = qP.Filter("DueDate >=", filters.StartTime)
				qP = qP.Filter("DueDate <=", filters.EndTime)
			}
			if filters.AgentID != "" {
				qP = qP.Filter("AgentID =", filters.AgentID)
			}
			if global.Options.DSName != "" {
				qP = qP.Namespace(global.Options.DSName)
			}
			qP.Filter("Company=", company)
			cursor1 := cursors[1]
			if cursor1 != "" {
				cursor, err := datastore.DecodeCursor(cursor1)
				if err != nil {
					log.Printf("Bad cursor %q: %v", cursor, err)
				}
				qP = qP.Start(cursor)
			}
			iteratorDental := db.client.Run(ctx, qP)
			for {
				var dentalOne contracts.PatientDentalInsurance
				_, err := iteratorDental.Next(&dentalOne)
				if err != nil {
					log.Printf("query ended %q: %v", cursor, err)
					break
				}
				dentalInsurance = append(dentalInsurance, dentalOne)
				// Do something with the Person p
			}
			nextCursor, err := iteratorDental.Cursor()
			if err != nil {
				mainCursor += "cursor_" + ""
			} else {
				mainCursor += "cursor_" + nextCursor.String()
			}
			qP = datastore.NewQuery("MedicalInsuranceIndexed").Limit(pageSize)
			qP = qP.Filter("AddressID=", addressID)
			if filters.StartTime > 0 && filters.EndTime > 0 {
				qP = qP.Filter("DueDate >=", filters.StartTime)
				qP = qP.Filter("DueDate <=", filters.EndTime)
			}
			if filters.AgentID != "" {
				qP = qP.Filter("AgentID =", filters.AgentID)
			}
			if global.Options.DSName != "" {
				qP = qP.Namespace(global.Options.DSName)
			}
			qP.Filter("Company=", company)

			cursor2 := cursors[2]
			if cursor2 != "" {
				cursor, err := datastore.DecodeCursor(cursor2)
				if err != nil {
					log.Printf("Bad cursor %q: %v", cursor, err)
				}
				qP = qP.Start(cursor)
			}
			iteratorMedical := db.client.Run(ctx, qP)
			for {
				var medicalOne contracts.PatientMedicalInsurance
				_, err := iteratorMedical.Next(&medicalOne)
				if err != nil {
					log.Printf("query ended %q: %v", cursor, err)
					break
				}
				medicalInsurance = append(medicalInsurance, medicalOne)
				// Do something with the Person p
			}
			nextCursor, err = iteratorDental.Cursor()
			if err != nil {
				mainCursor += "cursor_" + ""
			} else {
				mainCursor += "cursor_" + nextCursor.String()
			}
			if len(medicalInsurance) > 0 {
				for _, insurance := range medicalInsurance {
					if patient, ok := patientsMap[insurance.PatientID]; ok {
						patient.MedicalInsurance = append(patient.MedicalInsurance, insurance)
						patientsMap[patient.PatientID] = patient
					} else {
						patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
						if err == nil {
							patientOne.MedicalInsurance = append(patientOne.MedicalInsurance, insurance)
							patientsMap[patientOne.PatientID] = *patientOne
						}
					}
				}
			}
			if len(dentalInsurance) > 0 {
				for _, insurance := range dentalInsurance {
					if patient, ok := patientsMap[insurance.PatientID]; ok {
						patient.DentalInsurance = append(patient.DentalInsurance, insurance)
						patientsMap[patient.PatientID] = patient
					} else {
						patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
						if err == nil {
							patientOne.DentalInsurance = append(patientOne.DentalInsurance, insurance)
							patientsMap[patientOne.PatientID] = *patientOne
						}
					}
				}
			}
		}
	} else {
		dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
		medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
		qP := datastore.NewQuery("DentalInsuranceIndexed").Limit(pageSize)
		qP = qP.Filter("AddressID=", addressID)
		if cursor == "" {
			cursor = "cursor_" + "cursor_"
		}
		cursors := strings.Split(cursor, "cursor_")
		if filters.StartTime > 0 && filters.EndTime > 0 {
			qP = qP.Filter("DueDate >=", filters.StartTime)
			qP = qP.Filter("DueDate <=", filters.EndTime)
		}
		if filters.AgentID != "" {
			qP = qP.Filter("AgentID =", filters.AgentID)
		}
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		cursor1 := cursors[1]
		if cursor1 != "" {
			cursor, err := datastore.DecodeCursor(cursor1)
			if err != nil {
				log.Printf("Bad cursor %q: %v", cursor, err)
			}
			qP = qP.Start(cursor)
		}
		iteratorDental := db.client.Run(ctx, qP)
		for {
			var dentalOne contracts.PatientDentalInsurance
			_, err := iteratorDental.Next(&dentalOne)
			if err != nil {
				log.Printf("query ended %q: %v", cursor, err)
				break
			}
			dentalInsurance = append(dentalInsurance, dentalOne)
			// Do something with the Person p
		}
		nextCursor, err := iteratorDental.Cursor()
		if err != nil {
			mainCursor += "cursor_" + ""
		} else {
			mainCursor += "cursor_" + nextCursor.String()
		}
		qP = datastore.NewQuery("MedicalInsuranceIndexed").Limit(pageSize)
		qP = qP.Filter("AddressID=", addressID)
		if filters.StartTime > 0 && filters.EndTime > 0 {
			qP = qP.Filter("DueDate >=", filters.StartTime)
			qP = qP.Filter("DueDate <=", filters.EndTime)
		}
		if filters.AgentID != "" {
			qP = qP.Filter("AgentID =", filters.AgentID)
		}
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		cursor2 := cursors[2]
		if cursor2 != "" {
			cursor, err := datastore.DecodeCursor(cursor2)
			if err != nil {
				log.Printf("Bad cursor %q: %v", cursor, err)
			}
			qP = qP.Start(cursor)
		}
		iteratorMedical := db.client.Run(ctx, qP)
		for {
			var medicalOne contracts.PatientMedicalInsurance
			_, err := iteratorMedical.Next(&medicalOne)
			if err != nil {
				log.Printf("query ended %q: %v", cursor, err)
				break
			}
			medicalInsurance = append(medicalInsurance, medicalOne)
			// Do something with the Person p
		}
		nextCursor, err = iteratorDental.Cursor()
		if err != nil {
			mainCursor += "cursor_" + ""
		} else {
			mainCursor += "cursor_" + nextCursor.String()
		}
		if len(medicalInsurance) > 0 {
			for _, insurance := range medicalInsurance {
				if patient, ok := patientsMap[insurance.PatientID]; ok {
					patient.MedicalInsurance = append(patient.MedicalInsurance, insurance)
					patientsMap[patient.PatientID] = patient
				} else {
					patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
					if err == nil {
						patientOne.MedicalInsurance = append(patientOne.MedicalInsurance, insurance)
						patientsMap[patientOne.PatientID] = *patientOne
					}
				}
			}
		}
		if len(dentalInsurance) > 0 {
			for _, insurance := range dentalInsurance {
				if patient, ok := patientsMap[insurance.PatientID]; ok {
					patient.DentalInsurance = append(patient.DentalInsurance, insurance)
					patientsMap[patient.PatientID] = patient
				} else {
					patientOne, err := db.GetPatientByAgentInsurances(ctx, insurance.PatientID)
					if err == nil {
						patientOne.DentalInsurance = append(patientOne.DentalInsurance, insurance)
						patientsMap[patientOne.PatientID] = *patientOne
					}
				}
			}
		}
	}
	patients := make([]contracts.Patient, 0)

	for _, patient := range patientsMap {
		patients = append(patients, patient)
	}
	return patients, mainCursor
}

// GetPatientByAddressID ...
func (db DSPatient) GetPatientByAddressID(ctx context.Context, addressID string) []contracts.PatientStore {
	patients := make([]contracts.PatientStore, 0)
	patients1 := make([]contracts.PatientStore, 0)
	patients2 := make([]contracts.PatientStore, 0)
	patients3 := make([]contracts.PatientStore, 0)
	qP := datastore.NewQuery("PatientIndexed")
	qP = qP.Filter("GD =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients1)
	qP = datastore.NewQuery("PatientIndexed")
	qP = qP.Filter("SP =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients2)
	qP = datastore.NewQuery("PatientIndexed")
	qP = qP.Filter("AddressID =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients3)
	if len(patients1) > 0 {
		patients = append(patients, patients1...)
	}
	if len(patients2) > 0 {
		patients = append(patients, patients2...)
	}
	if len(patients3) > 0 {
		patients = append(patients, patients3...)
	}
	return patients
}

// GetPatientByAddressIDPaginate ...
func (db DSPatient) GetPatientByAddressIDPaginate(ctx context.Context, addressID string, pageSize int, cursor string) (map[string]contracts.PatientStore, string) {
	patients := make(map[string]contracts.PatientStore, 0)
	if cursor == "" {
		cursor = "cursor_" + "cursor_" + "cursor_"
	}
	cursors := strings.Split(cursor, "cursor_")
	mainCursor := ""
	qP := datastore.NewQuery("PatientIndexed").Limit(pageSize)
	qP = qP.Filter("GD =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor1 := cursors[1]
	if cursor1 != "" {
		cursor, err := datastore.DecodeCursor(cursor1)
		if err != nil {
			log.Printf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient1 := db.client.Run(ctx, qP)
	for {
		var onePatient contracts.PatientStore
		_, err := iteratorPatient1.Next(&onePatient)
		if err != nil {
			log.Printf("query ended %q: %v", cursor, err)
			break
		}
		patients[onePatient.PatientID] = onePatient
		// Do something with the Person p
	}
	nextCursor, err := iteratorPatient1.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	qP = datastore.NewQuery("PatientIndexed").Limit(pageSize)
	qP = qP.Filter("SP =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor2 := cursors[2]
	if cursor2 != "" {
		cursor, err := datastore.DecodeCursor(cursor2)
		if err != nil {
			log.Printf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient2 := db.client.Run(ctx, qP)
	for {
		var onePatient contracts.PatientStore
		_, err := iteratorPatient2.Next(&onePatient)
		if err != nil {
			log.Printf("query ended %q: %v", cursor, err)
			break
		}
		patients[onePatient.PatientID] = onePatient
		// Do something with the Person p
	}
	nextCursor, err = iteratorPatient2.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	qP = datastore.NewQuery("PatientIndexed").Limit(pageSize)
	qP = qP.Filter("AddressID =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor3 := cursors[3]
	if cursor3 != "" {
		cursor, err := datastore.DecodeCursor(cursor3)
		if err != nil {
			log.Printf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient3 := db.client.Run(ctx, qP)
	for {
		var onePatient contracts.PatientStore
		_, err := iteratorPatient3.Next(&onePatient)
		if err != nil {
			log.Printf("query ended %q: %v", cursor, err)
			break
		}
		if onePatient.PatientID == "2a4f4d8f-925f-11eb-93a0-3293553715c5" {
			log.Println("2a4f4d8f-925f-11eb-93a0-3293553715c5")
		}
		patients[onePatient.PatientID] = onePatient
		// Do something with the Person p
	}
	nextCursor, err = iteratorPatient3.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	return patients, mainCursor
}
func (db DSPatient) ReturnPatientsWithDMInsurances(ctx context.Context, patientStores map[string]contracts.PatientStore) []contracts.Patient {
	patients := make([]contracts.Patient, 0)
	for _, patientData := range patientStores {
		if patientData.PatientID == "2a4f4d8f-925f-11eb-93a0-3293553715c5" {
			log.Println("2a4f4d8f-925f-11eb-93a0-3293553715c5")
		}
		patient := db.ParsePatient(ctx, patientData)
		patients = append(patients, patient)
	}
	return patients
}

func (db DSPatient) ReturnPatientsWithDMInsurancesArr(ctx context.Context, patientStores []contracts.PatientStore) []contracts.Patient {
	patients := make([]contracts.Patient, 0)
	for _, patientData := range patientStores {
		if patientData.PatientID == "2a4f4d8f-925f-11eb-93a0-3293553715c5" {
			log.Println("2a4f4d8f-925f-11eb-93a0-3293553715c5")
		}
		patient := db.ParsePatient(ctx, patientData)
		patients = append(patients, patient)
	}
	return patients
}
func (db DSPatient) ParsePatient(ctx context.Context, patientData contracts.PatientStore) contracts.Patient {
	var patientStore contracts.Patient
	patientStore.AddressID = patientData.AddressID
	patientStore.ClinicName = patientData.ClinicName
	patientStore.FirstName = patientData.FirstName
	patientStore.LastName = patientData.LastName
	patientStore.Dob = patientData.Dob
	patientStore.Email = patientData.Email
	patientStore.GD = patientData.GD
	patientStore.GDName = patientData.GDName
	patientStore.SP = patientData.SP
	patientStore.SPName = patientData.SPName
	patientStore.SSN = patientData.SSN
	patientStore.SameDay = patientData.SameDay
	patientStore.Phone = patientData.Phone
	patientStore.SSN = patientData.SSN
	patientStore.ZipCode = patientData.ZipCode
	patientStore.Status = patientData.Status
	patientStore.DueDate = patientData.DueDate
	patientStore.AppointmentTime = patientData.AppointmentTime
	patientStore.CreatedOn = patientData.CreatedOn
	patientStore.CreationDate = patientData.CreationDate
	patientStore.PatientID = patientData.PatientID
	for _, id := range patientData.DentalInsuraceID {
		insurance := db.GetDentalInsurance(ctx, id)
		if insurance.ID != "" {
			patientStore.DentalInsurance = append(patientStore.DentalInsurance, insurance)
		}
	}
	for _, id := range patientData.MedicalInsuranceID {
		insurance := db.GetMedicalInsurance(ctx, id)
		if insurance.ID != "" {
			patientStore.MedicalInsurance = append(patientStore.MedicalInsurance, insurance)
		}
	}
	return patientStore
}
func (db DSPatient) GetDentalInsurance(ctx context.Context, dID string) contracts.PatientDentalInsurance {
	var patientDental contracts.PatientDentalInsurance
	dIS := make([]contracts.PatientDentalInsurance, 0)

	qP := datastore.NewQuery("DentalInsuranceIndexed")
	qP = qP.Filter("ID =", dID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	_, err := db.client.GetAll(ctx, qP, &dIS)
	if err != nil {
		return patientDental
	}
	if len(dIS) > 0 {
		patientDental = dIS[0]
	}

	return patientDental
}

func (db DSPatient) GetMedicalInsurance(ctx context.Context, mID string) contracts.PatientMedicalInsurance {
	var patientMedical contracts.PatientMedicalInsurance
	mIS := make([]contracts.PatientMedicalInsurance, 0)

	qP := datastore.NewQuery("MedicalInsuranceIndexed")
	qP = qP.Filter("ID =", mID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	_, err := db.client.GetAll(ctx, qP, &mIS)
	if err != nil {
		return patientMedical
	}
	if len(mIS) > 0 {
		patientMedical = mIS[0]
	}

	return patientMedical
}

// GetPatientByID ...
func (db DSPatient) GetPatientByID(ctx context.Context, pID string) (*contracts.Patient, *contracts.PatientStore, *datastore.Key, error) {
	patients := make([]contracts.PatientStore, 0)
	var patientReturn contracts.Patient
	qP := datastore.NewQuery("PatientIndexed")
	qP = qP.Filter("PatientID =", pID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keys, err := db.client.GetAll(ctx, qP, &patients)
	if err != nil {
		return nil, nil, nil, err
	}
	patient := patients[0]
	key := keys[0]
	patientReturn = db.ParsePatient(ctx, patient)
	return &patientReturn, &patient, key, nil
}

// GetPatientByFilters ...
func (db DSPatient) GetPatientByAgentInsurances(ctx context.Context, pID string) (*contracts.Patient, error) {
	patients := make([]contracts.PatientStore, 0)
	qP := datastore.NewQuery("PatientIndexed")
	qP = qP.Filter("PatientID =", pID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	_, err := db.client.GetAll(ctx, qP, &patients)
	if err != nil {
		return nil, err
	}
	patientData := patients[0]
	var patientStore contracts.Patient
	patientStore.AddressID = patientData.AddressID
	patientStore.ClinicName = patientData.ClinicName
	patientStore.FirstName = patientData.FirstName
	patientStore.LastName = patientData.LastName
	patientStore.Dob = patientData.Dob
	patientStore.Email = patientData.Email
	patientStore.GD = patientData.GD
	patientStore.GDName = patientData.GDName
	patientStore.SP = patientData.SP
	patientStore.SPName = patientData.SPName
	patientStore.SSN = patientData.SSN
	patientStore.SameDay = patientData.SameDay
	patientStore.Phone = patientData.Phone
	patientStore.SSN = patientData.SSN
	patientStore.ZipCode = patientData.ZipCode
	patientStore.Status = patientData.Status
	patientStore.DueDate = patientData.DueDate
	patientStore.AppointmentTime = patientData.AppointmentTime
	patientStore.CreatedOn = patientData.CreatedOn
	patientStore.CreationDate = patientData.CreationDate
	patientStore.PatientID = patientData.PatientID
	return &patientStore, nil
}

// UpdatePatientStatus .....
func (db DSPatient) UpdateInsuranceStatus(ctx context.Context, pID string, status contracts.PatientStatus) error {
	dInsurance := db.GetDentalInsurance(ctx, pID)
	if dInsurance.ID != "" {
		dInsurance.Status = status
		pKey := datastore.NameKey("DentalInsuranceIndexed", dInsurance.ID, nil)
		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &dInsurance)
		if err != nil {
			return err
		}
		return nil
	}
	mInsurance := db.GetMedicalInsurance(ctx, pID)
	if mInsurance.ID != "" {
		mInsurance.Status = status
		pKey := datastore.NameKey("MedicalInsuranceIndexed", mInsurance.ID, nil)
		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &mInsurance)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("insurance not found")
}

// UpdatePatientStatus ....
func (db DSPatient) AddAgentToInsurance(ctx context.Context, pID string, agent string) error {
	dInsurance := db.GetDentalInsurance(ctx, pID)
	if dInsurance.ID != "" {
		dInsurance.AgentID = agent
		pKey := datastore.NameKey("DentalInsuranceIndexed", dInsurance.ID, nil)
		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &dInsurance)
		if err != nil {
			return err
		}
		return nil
	}
	mInsurance := db.GetMedicalInsurance(ctx, pID)
	if mInsurance.ID != "" {
		mInsurance.AgentID = agent
		pKey := datastore.NameKey("MedicalInsuranceIndexed", mInsurance.ID, nil)
		if global.Options.DSName != "" {
			pKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, pKey, &mInsurance)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("insurance not found")
}

// UpdatePatientStatus ....
func (db DSPatient) ListInsuranceCompanies(ctx context.Context) ([]string, error) {
	qP := datastore.NewQuery("DentalInsuranceIndexed").Project("Company").DistinctOn("Company")
	qP = qP.Namespace("sdprod")
	it := db.client.Run(ctx, qP)
	companies := make(map[string]string, 0)
	for {
		var dentalOne contracts.PatientDentalInsurance
		if _, err := it.Next(&dentalOne); err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("end of companies")
		}
		comp := strings.TrimSpace(dentalOne.Company)
		companies[comp] = comp
	}
	qP = datastore.NewQuery("MedicalInsuranceIndexed").Project("Company").DistinctOn("Company")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	it = db.client.Run(ctx, qP)
	for {
		var medOne contracts.PatientMedicalInsurance
		if _, err := it.Next(&medOne); err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("end of companies")
		}
		comp := strings.TrimSpace(medOne.Company)
		companies[comp] = comp
	}
	returnedCompanies := make([]string, 0)
	for _, value := range companies {
		returnedCompanies = append(returnedCompanies, value)
	}
	return returnedCompanies, nil
}
