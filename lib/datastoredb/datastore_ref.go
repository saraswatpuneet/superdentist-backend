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

// DSReferral ...
type DSReferral struct {
	projectID string
	client    *datastore.Client
}

//NewReferralHandler return new database action
func NewReferralHandler() *DSReferral {
	return &DSReferral{projectID: "", client: nil}
}

// InitializeDataBase ....
func (db *DSReferral) InitializeDataBase(ctx context.Context, projectID string) error {
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

// CreateReferral .....
func (db *DSReferral) CreateReferral(ctx context.Context, referral contracts.DSReferral) error {
	primaryKey := datastore.NameKey("ClinicReferrals", referral.ReferralID, nil)
	//lets create the clinic
	_, err := db.client.Put(ctx, primaryKey, &referral)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// GetReferral .....
func (db *DSReferral) GetReferral(ctx context.Context, refID string) (*contracts.DSReferral, error) {
	primaryKey := datastore.NameKey("ClinicReferrals", refID, nil)
	var referral contracts.DSReferral
	err := db.client.Get(ctx, primaryKey, &referral)
	if err != nil {
		return nil, err
	}
	return &referral, nil
}

// DeleteReferral .....
func (db *DSReferral) DeleteReferral(ctx context.Context, refID string) (*contracts.DSReferral, error) {
	primaryKey := datastore.NameKey("ClinicReferrals", refID, nil)
	var referral contracts.DSReferral
	err := db.client.Delete(ctx, primaryKey)
	if err != nil {
		return nil, err
	}
	return &referral, nil
}
