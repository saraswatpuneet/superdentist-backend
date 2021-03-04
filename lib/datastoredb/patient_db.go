package datastoredb

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
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
func (db DSPatient) AddPatientInformation(ctx context.Context, patient contracts.Patient) (string, error) {
	patientID, _ := uuid.NewUUID()
	pIDString := patientID.String()
	pKey := datastore.NameKey("PatientDetails", pIDString, nil)
	if global.Options.DSName != "" {
		pKey.Namespace = global.Options.DSName
	}
	_, err := db.client.Put(ctx, pKey, &patient)
	if err != nil {
		return "", err
	}
	return pIDString, nil
}
