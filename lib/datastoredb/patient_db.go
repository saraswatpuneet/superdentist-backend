package datastoredb

import (
	"context"
	"fmt"
	"os"

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
func (db DSPatient) AddPatientInformation(ctx context.Context, patient contracts.Patient, pIDString string) (string, error) {

	pKey := datastore.NameKey("PatientDetails", pIDString, nil)
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
	pKey := datastore.NameKey("PatientNotes", notes.PatientID, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	_, err := db.client.Put(ctx, pKey, &notes)
	if err != nil {
		return err
	}
	return nil
}

// GetPatientByAddressID ...
func (db DSPatient) GetPatientByAddressID(ctx context.Context, addressID string) []contracts.Patient {
	patients := make([]contracts.Patient, 0)
	patients1 := make([]contracts.Patient, 0)
	patients2 := make([]contracts.Patient, 0)
	patients3 := make([]contracts.Patient, 0)

	qP := datastore.NewQuery("PatientDetails")
	qP = qP.Filter("GD =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients1)
	qP = datastore.NewQuery("PatientDetails")
	qP = qP.Filter("SP =", addressID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	db.client.GetAll(ctx, qP, &patients2)
	qP = datastore.NewQuery("PatientDetails")
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

// GetPatientByID ...
func (db DSPatient) GetPatientByID(ctx context.Context, pID string) (*contracts.Patient, *datastore.Key, error) {
	patients := make([]contracts.Patient, 0)

	qP := datastore.NewQuery("PatientDetails")
	qP = qP.Filter("PatientID =", pID)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keys, err := db.client.GetAll(ctx, qP, &patients)
	if err != nil {
		return nil, nil, err
	}
	patient := patients[0]
	key := keys[0]
	return &patient, key, nil
}

// UpdatePatientStatus
func (db DSPatient) UpdatePatientStatus(ctx context.Context, pID string, status string) error {
	patient, _, err := db.GetPatientByID(ctx, pID)
	if err != nil {
		return err
	}
	patient.Status = status
	_, err = db.AddPatientInformation(ctx, *patient, pID)
	if err != nil {
		return err
	}
	return nil
}
