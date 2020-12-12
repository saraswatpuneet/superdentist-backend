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

type dsClinic struct {
	projectID string
	client    *datastore.Client
}

//NewClinicHandler return new database action
func NewClinicHandler() *dsClinic {
	return &dsClinic{projectID: "", client: nil}
}

// Ensure dsClinic conforms to the ClinicRegistrationDatabase interface.

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
func (db *dsClinic) AddClinicRegistration(ctx context.Context, clinic *contracts.ClinicRegistrationData, uID string) error {
	parentKey := datastore.NameKey("ClinicAdmin", uID, nil)
	if global.Options.DSName!= "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinic.EmailID, parentKey)
	if global.Options.DSName!= "" {
		primaryKey.Namespace = global.Options.DSName
	}	
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	qP := datastore.NewQuery("ClinicAdmin").Ancestor(primaryKey)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keyClinics, err := db.client.GetAll(ctx, qP, allPrimaryClinics)
	noKeys := len(keyClinics)
	if err != nil || noKeys <= 0 {
		//lets create the clinic
		_, err := db.client.Put(ctx, primaryKey, clinic)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		return nil
	}
	return fmt.Errorf("cannot register the admin as it is already registred with same credentials: %v", err)

}

// VerifyClinicInDatastore ..
func (db *dsClinic) VerifyClinicInDatastore(ctx context.Context, emailID string, uID string) error {
	parentKey := datastore.NameKey("ClinicAdmin", uID, nil)
	if global.Options.DSName!= "" {
		parentKey.Namespace = global.Options.DSName
	}
	pk := datastore.NameKey("ClinicAdmin", emailID, parentKey)
	if global.Options.DSName!= "" {
		pk.Namespace = global.Options.DSName
	}
	clinic := &contracts.ClinicRegistrationData{}
	if err := db.client.Get(ctx, pk, clinic); err != nil {
		return fmt.Errorf("datastoredb: could not get registered cli: %v", err)

	}
	clinic.IsVerified = true
	_, err := db.client.Put(ctx, pk, clinic)
	if err != nil {
		return fmt.Errorf("datastoredb: could not verify clinic: %v", err)
	}
	return nil
}

// Close closes the database.
func (db *dsClinic) Close() error {
	return db.client.Close()
}
