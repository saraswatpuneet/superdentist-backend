package datastoredb

import (
	"context"
	"fmt"
	"os"

	guuid "github.com/google/uuid"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
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

func (db dsClinicMeta) AddPhysicalAddessressToClinic(ctx context.Context, clinicEmailID string, clinicFBID string, addresses []contracts.PhysicalClinicsRegistration, mapsClient *gmaps.ClientGMaps) ([]contracts.PhysicalClinicsRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedAddress := make([]contracts.PhysicalClinicsRegistration, 0)
	for _, address := range addresses {
		addrID, err := guuid.NewUUID()
		address.ClinicAddressID = addrID.String()
		if err != nil {
			return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		gmapAddress, err := mapsClient.FindPlacesFromText(address.ClinicAddress)
		location := contracts.ClinicLocation{
			Lat:  0.0,
			Long: 0.0,
		}
		if err == nil && len(gmapAddress.Results) > 0 {
			currentLocation := gmapAddress.Results[0]
			location.Lat = currentLocation.Geometry.Location.Lat
			location.Long = currentLocation.Geometry.Location.Lng
		}
		addressKey := datastore.NameKey("ClinicAddress", addrID.String(), primaryKey)
		currentLocWithMap := contracts.PhysicalClinicMapLocation{
			PhysicalClinicsRegistration: address,
			Location:                    location,
		}
		_, err = db.client.Put(ctx, addressKey, &currentLocWithMap)
		if err != nil {
			return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		returnedAddress = append(returnedAddress, address)
	}
	return returnedAddress, nil
}

// AddDoctorsToPhysicalClincs ....
func (db dsClinicMeta) AddDoctorsToPhysicalClincs(ctx context.Context, clinicEmailID string, clinicFBID string, doctorsData []contracts.ClinicDoctorsDetails) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	for _, doctor := range doctorsData {
		for _, doc := range doctor.Doctors {
			docID, err := guuid.NewUUID()
			doc.ClinicAddressID = doctor.ClinicAddressID
			clinicDoctorKey := datastore.NameKey("ClinicDoctors", docID.String(), primaryKey)
			if err != nil {
				return fmt.Errorf("cannot register doctor with sd: %v", err)
			}
			_, err = db.client.Put(ctx, clinicDoctorKey, &doc)
			if err != nil {
				return fmt.Errorf("cannot register doctor with sd: %v", err)
			}
		}
	}
	return nil
}

// AddPMSUsedByClinics ......
func (db *dsClinicMeta) AddPMSUsedByClinics(ctx context.Context, clinicEmailID string, clinicFBID string, pmsData []string) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	clinicPMSKey := datastore.IncompleteKey("ClinicPMS", primaryKey)
	_, err := db.client.Put(ctx, clinicPMSKey, &pmsData)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// AddServicesForClinic .....
func (db *dsClinicMeta) AddServicesForClinic(ctx context.Context, clinicEmailID string, clinicFBID string, serviceData []contracts.ServiceObject) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	clinicPMSKey := datastore.IncompleteKey("ClinicServices", primaryKey)
	_, err := db.client.Put(ctx, clinicPMSKey, serviceData)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// GetAllClinics ....
func (db *dsClinicMeta) GetAllClinics(ctx context.Context, clinicEmailID string, clinicFBID string) ([]contracts.PhysicalClinicsRegistration, string, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedAddress := make([]contracts.PhysicalClinicsRegistration, 0)
	qP := datastore.NewQuery("ClinicDoctors").Ancestor(primaryKey)
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddress)
	if err != nil || len(keysClinics) <= 0 {
		return nil, "", fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	/// Get Admin clinic type
	qP = datastore.NewQuery("ClinicAdmin").Ancestor(primaryKey)
	allPrimaryClinics := make([]contracts.ClinicRegistrationData, 0)
	_, err = db.client.GetAll(ctx, qP, allPrimaryClinics)
	if err != nil || len(keysClinics) <= 0 {
		return nil, "", fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return returnedAddress, allPrimaryClinics[0].ClinicType, nil
}

// GetClinicDoctors ....
func (db *dsClinicMeta) GetClinicDoctors(ctx context.Context, clinicEmailID string, clinicFBID string, clinicAddressID string) ([]contracts.ClinicDoctorRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedDoctors := make([]contracts.ClinicDoctorRegistration, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if clinicAddressID != "" {
		qP = qP.Filter("ClinicAddressID =", clinicAddressID)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedDoctors)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	return returnedDoctors, nil
}

// Close closes the database.
func (db *dsClinicMeta) Close() error {
	return db.client.Close()
}
