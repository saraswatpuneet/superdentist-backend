package datastoredb

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"

	"cloud.google.com/go/datastore"
)

type dsClinicMeta struct {
	projectID string
	client    *datastore.Client
}

//NewClinicMetaHandler return new database action
func NewClinicMetaHandler() *dsClinicMeta {
	return &dsClinicMeta{projectID: "", client: nil}
}

// Ensure dsClinic conforms to the ClinicPhysicalAddressDatabase interface.

var _ contracts.ClinicPhysicalAddressDatabase = &dsClinicMeta{}

// InitializeDataBase ....
func (db *dsClinicMeta) InitializeDataBase(ctx context.Context, projectID string) error {
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

func (db dsClinicMeta) AddPhysicalAddessressToClinic(ctx context.Context, clinicEmailID string, clinicFBID string, addresses []contracts.PhysicalClinicsRegistration) ([]contracts.PhysicalClinicsRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	qP := datastore.NewQuery("ClinicAdmin").Ancestor(primaryKey)
	returnedAddress := make([]contracts.PhysicalClinicsRegistration, 0)
	keyClinics, err := db.client.GetAll(ctx, qP, allPrimaryClinics)
	if err != nil {
		return nil, err
	}
	noKeys := len(keyClinics)
	if noKeys > 0 && noKeys < 2 {
		currentPrimarykey := keyClinics[0]
		for _, address := range addresses {
			addressKey := datastore.IncompleteKey("ClinicAddress", currentPrimarykey)
			registredKey, err := db.client.Put(ctx, addressKey, address)
			if err != nil {
				return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
			}
			uniqueAddressID := registredKey.ID
			address.ClinicAddressID = strconv.FormatInt(uniqueAddressID, 10)
			returnedAddress = append(returnedAddress, address)
		}
		return returnedAddress, nil
	}
	return nil, fmt.Errorf("Bad database entry, more than one entity exists for admin")
}

// AddDoctorsToPhysicalClincs ....
func (db dsClinicMeta) AddDoctorsToPhysicalClincs(ctx context.Context, clinicEmailID string, clinicFBID string, doctorsData []contracts.ClinicDoctorsDetails) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	qP := datastore.NewQuery("ClinicAdmin").Ancestor(primaryKey)
	keyClinics, err := db.client.GetAll(ctx, qP, allPrimaryClinics)
	if err != nil {
		return err
	}
	noKeys := len(keyClinics)
	if noKeys > 0 && noKeys < 2 {
		currentPrimarykey := keyClinics[0]
		for _, doctor := range doctorsData {
			clinicDoctorKey := datastore.NameKey("ClinicDoctors", doctor.ClinicAddressID, currentPrimarykey)
			_, err := db.client.Put(ctx, clinicDoctorKey, doctor.Doctors)
			if err != nil {
				return fmt.Errorf("cannot register clinic with sd: %v", err)
			}
		}
		return nil
	}
	return fmt.Errorf("Bad database entry, more than one entity exists for admin")
}

func (db *dsClinicMeta) AddPMSUsedByClinics(ctx context.Context, clinicEmailID string, clinicFBID string, pmsData []string) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	qP := datastore.NewQuery("ClinicAdmin").Ancestor(primaryKey)
	keyClinics, err := db.client.GetAll(ctx, qP, allPrimaryClinics)
	if err != nil {
		return err
	}
	noKeys := len(keyClinics)
	if noKeys > 0 && noKeys < 2 {
		currentPrimarykey := keyClinics[0]
		clinicPMSKey := datastore.IncompleteKey("ClinicPMS", currentPrimarykey)
		_, err := db.client.Put(ctx, clinicPMSKey, pmsData)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		return nil
	}
	return fmt.Errorf("Bad database entry, more than one entity exists for admin")
}

// Close closes the database.
func (db *dsClinicMeta) Close() error {
	return db.client.Close()
}
