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

	pKey := datastore.NameKey("Patient", pIDString, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	patient.PatientID = pIDString
	_, err := db.client.Put(ctx, pKey, &patient)
	if err != nil {
		return "", err
	}
	for _, insurance := range dI {
		pKey := datastore.NameKey("DentalInsurance", insurance.ID, nil)
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
		pKey := datastore.NameKey("MedicalInsurance", insurance.ID, nil)
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

	pKey := datastore.NameKey("Patient", pIDString, nil)
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
	dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
	medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
	qP := datastore.NewQuery("DentalInsurance")
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

	qP = datastore.NewQuery("MedicalInsurance")
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
	patientsMap := make(map[string]contracts.Patient, 0)
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
	patients := make([]contracts.Patient, 0)
	for _, patient := range patientsMap {
		patients = append(patients, patient)
	}
	return patients
}

// GetPatientByFiltersPaginate ...
func (db DSPatient) GetPatientByFiltersPaginate(ctx context.Context, addressID string, filters contracts.PatientFilters, pageSize int, cursor string) ([]contracts.Patient, string) {
	dentalInsurance := make([]contracts.PatientDentalInsurance, 0)
	medicalInsurance := make([]contracts.PatientMedicalInsurance, 0)
	qP := datastore.NewQuery("DentalInsurance").Limit(pageSize)
	if cursor == "" {
		cursor = "cursor_" + "cursor_"
	}
	cursors := strings.Split(cursor, "cursor_")
	mainCursor := ""
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
			log.Fatalf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorDental := db.client.Run(ctx, qP)
	var dentalOne contracts.PatientDentalInsurance
	_, err := iteratorDental.Next(&dentalOne)
	for err == nil {
		dentalInsurance = append(dentalInsurance, dentalOne)
		_, err = iteratorDental.Next(&dentalOne)
	}
	nextCursor, err := iteratorDental.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	qP = datastore.NewQuery("MedicalInsurance").Limit(pageSize)
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
			log.Fatalf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorMedical := db.client.Run(ctx, qP)
	var medicalOne contracts.PatientMedicalInsurance
	_, err = iteratorMedical.Next(&medicalOne)
	for err == nil {
		medicalInsurance = append(medicalInsurance, medicalOne)
		_, err = iteratorDental.Next(&dentalOne)
	}
	nextCursor, err = iteratorDental.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	patientsMap := make(map[string]contracts.Patient, 0)
	patients := make([]contracts.Patient, 0)
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
	qP := datastore.NewQuery("Patient")
	qP = qP.Filter("GD =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients1)
	qP = datastore.NewQuery("Patient")
	qP = qP.Filter("SP =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients2)
	qP = datastore.NewQuery("Patient")
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
func (db DSPatient) GetPatientByAddressIDPaginate(ctx context.Context, addressID string, pageSize int, cursor string) ([]contracts.PatientStore, string) {
	patients := make([]contracts.PatientStore, 0)
	if cursor == "" {
		cursor = "cursor_" + "cursor_" + "cursor_"
	}
	cursors := strings.Split(cursor, "cursor_")
	mainCursor := ""
	qP := datastore.NewQuery("Patient").Limit(pageSize)
	qP = qP.Filter("GD =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor1 := cursors[1]
	if cursor1 != "" {
		cursor, err := datastore.DecodeCursor(cursor1)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient := db.client.Run(ctx, qP)
	var onePatient contracts.PatientStore
	_, err := iteratorPatient.Next(&onePatient)
	for err == nil {
		patients = append(patients, onePatient)
		_, err = iteratorPatient.Next(&onePatient)
	}
	nextCursor, err := iteratorPatient.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	qP = datastore.NewQuery("Patient").Limit(pageSize)
	qP = qP.Filter("SP =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor2 := cursors[2]
	if cursor2 != "" {
		cursor, err := datastore.DecodeCursor(cursor2)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient = db.client.Run(ctx, qP)
	_, err = iteratorPatient.Next(&onePatient)
	for err == nil {
		patients = append(patients, onePatient)
		_, err = iteratorPatient.Next(&onePatient)
	}
	nextCursor, err = iteratorPatient.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	qP = datastore.NewQuery("Patient").Limit(pageSize)
	qP = qP.Filter("AddressID =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	cursor3 := cursors[3]
	if cursor3 != "" {
		cursor, err := datastore.DecodeCursor(cursor3)
		if err != nil {
			log.Fatalf("Bad cursor %q: %v", cursor, err)
		}
		qP = qP.Start(cursor)
	}
	iteratorPatient = db.client.Run(ctx, qP)
	_, err = iteratorPatient.Next(&onePatient)
	for err == nil {
		patients = append(patients, onePatient)
		_, err = iteratorPatient.Next(&onePatient)
	}
	nextCursor, err = iteratorPatient.Cursor()
	if err != nil {
		mainCursor += "cursor_" + ""
	} else {
		mainCursor += "cursor_" + nextCursor.String()
	}
	return patients, mainCursor
}
func (db DSPatient) ReturnPatientsWithDMInsurances(ctx context.Context, patientStores []contracts.PatientStore) []contracts.Patient {
	patients := make([]contracts.Patient, 0)
	for _, patientData := range patientStores {
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

	qP := datastore.NewQuery("DentalInsurance")
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

	qP := datastore.NewQuery("MedicalInsurance")
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
	qP := datastore.NewQuery("Patient")
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
	qP := datastore.NewQuery("Patient")
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
		pKey := datastore.NameKey("DentalInsurance", dInsurance.ID, nil)
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
		pKey := datastore.NameKey("MedicalInsurance", mInsurance.ID, nil)
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
		pKey := datastore.NameKey("DentalInsurance", dInsurance.ID, nil)
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
		pKey := datastore.NameKey("MedicalInsurance", mInsurance.ID, nil)
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
