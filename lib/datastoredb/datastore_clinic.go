package datastoredb

import (
	"context"
	"fmt"
	"os"

	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"

	"cloud.google.com/go/datastore"
)

type dsClinic struct {
	projectID string
	client    *datastore.Client
}

//NewClinicHandler return new database action
func NewClinicHandler() *dsClinic {
	return &dsClinic{projectID: "", client: nil}
}

// Ensure dsClinic conforms to the ComputeActionDatabase interface.

var _ contracts.ClinicRegistrationDatabase = &dsClinic{}

// InitializeDataBase ....
func (db *dsClinic) InitializeDataBase(ctx context.Context, projectID string) error {
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

// AddClinicRegistration ....
func (db *dsClinic) AddClinicRegistration(ctx context.Context, clinic *contracts.ClinicRegistrationData) (int64, error) {
	primaryKey := datastore.NameKey("ClinicRegistrationMain", clinic.EmailID, nil)
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	qP := datastore.NewQuery("ClinicRegistrationMain").Ancestor(primaryKey)

	keyClinics, err := db.client.GetAll(ctx, qP, allPrimaryClinics)
	noKeys := len(keyClinics)
	if err != nil || noKeys <= 0 {
		//lets create the clinic
		k := datastore.IncompleteKey("ClinicRegistrationMain", primaryKey)
		k, err = db.client.Put(ctx, k, clinic)
		if err != nil {
			return 0, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		clinic.ClinicID = k.ID
		return k.ID, nil
	}
	return 0, fmt.Errorf("cannot register the clinic as it is already registred with same credentials: %v", err)

}

// VerifyClinicInDatastore ..
func (db *dsClinic) VerifyClinicInDatastore(ctx context.Context, emailID string) (int64, error) {
	pk := datastore.NameKey("ClinicRegistrationMain", emailID, nil)
	clinic := &contracts.ClinicRegistrationData{}
	if err := db.client.Get(ctx, pk, clinic); err != nil {
		return 0.0, fmt.Errorf("datastoredb: could not get registered cli: %v", err)

	}
	clinic.IsVerified = true
	returnedKey, err := db.client.Put(ctx, pk, clinic)
	if err != nil {
		return 0.0, fmt.Errorf("datastoredb: could not verify clinic: %v", err)
	}
	return returnedKey.ID, nil
}

// Close closes the database.
func (db *dsClinic) Close() error {
	return db.client.Close()
}