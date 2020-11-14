package datastoredb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
	guuid "github.com/google/uuid"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/geohash"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"
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
		address.AddressID = addrID.String()
		if err != nil {
			return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		gmapAddress, err := mapsClient.FindPlacesFromText(address.Address)
		location := contracts.ClinicLocation{
			Lat:  0.0,
			Long: 0.0,
		}
		placeID := ""
		if err == nil && len(gmapAddress.Results) > 0 {
			currentLocation := gmapAddress.Results[0]
			location.Lat = currentLocation.Geometry.Location.Lat
			location.Long = currentLocation.Geometry.Location.Lng
			placeID = currentLocation.PlaceID
		}
		addressKey := datastore.NameKey("ClinicAddress", addrID.String(), primaryKey)
		currentHash := geohash.Encode(location.Lat, location.Long, 12)
		currentLocWithMap := contracts.PhysicalClinicMapLocation{
			PhysicalClinicsRegistration: address,
			Location:                    location,
			Geohash:                     currentHash,
			Precision:                   12,
			PlaceID:                     placeID,
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
			doc.AddressID = doctor.AddressID
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
	currentPMSStruct := contracts.PostPMSDetails{
		PMSNames: pmsData,
	}
	_, err := db.client.Put(ctx, clinicPMSKey, &currentPMSStruct)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// AddPMSAuthDetails ......
func (db *dsClinicMeta) AddPMSAuthDetails(ctx context.Context, clinicEmailID string, clinicFBID string, pmsInformation contracts.PostPMSAuthDetails) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	for _, pmsData := range pmsInformation.PMSAuthData {
		clinicPMSKey := datastore.NameKey("ClinicPMSAuth", pmsData.PMSName, primaryKey)
		bytesSafe, _ := json.Marshal(pmsData)
		var dsPMSAuth contracts.PMSAuthStructStore
		dsPMSAuth.PMSName = pmsData.PMSName
		sEnc := base64.StdEncoding.EncodeToString(bytesSafe)
		dsPMSAuth.AuthDetails = sEnc
		_, err := db.client.Put(ctx, clinicPMSKey, &dsPMSAuth)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
	}

	return nil
}

// AddServicesForClinic .....
func (db *dsClinicMeta) AddServicesForClinic(ctx context.Context, clinicEmailID string, clinicFBID string, serviceData []contracts.ServiceObject) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	for _, serObj := range serviceData {
		clinicPMSKey := datastore.IncompleteKey("ClinicServices", primaryKey)
		_, err := db.client.Put(ctx, clinicPMSKey, &serObj)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
	}

	return nil
}

// GetAllClinics ....
func (db *dsClinicMeta) GetAllClinics(ctx context.Context, clinicEmailID string, clinicFBID string) ([]contracts.PhysicalClinicMapLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedAddress := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddress)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return returnedAddress, nil
}

// GetClinicDoctors ....
func (db *dsClinicMeta) GetClinicDoctors(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string) ([]contracts.ClinicDoctorRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedDoctors := make([]contracts.ClinicDoctorRegistration, 0)
	qP := datastore.NewQuery("ClinicDoctors").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedDoctors)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	return returnedDoctors, nil
}

// GetSingleClinic ....
func (db *dsClinicMeta) GetSingleClinic(ctx context.Context, addressID string) (*contracts.PhysicalClinicMapLocation, error) {

	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress")
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return &returnedAddresses[0], nil
}

// GetNearbyClinics ....
func (db *dsClinicMeta) GetNearbyClinics(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string, distance float64) ([]contracts.PhysicalClinicMapLocation, *contracts.ClinicLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	returnedAddress := returnedAddresses[0]
	currentLatLong := returnedAddress.Location
	lat := 0.0144927536231884 // degrees latitude per mile
	lon := 0.0181818181818182 // degrees longitude per mile
	lowerLat := currentLatLong.Lat - lat*distance
	lowerLon := currentLatLong.Long - lon*distance

	upperLat := currentLatLong.Lat + lat*distance
	upperLon := currentLatLong.Long + lon*distance
	lowerHash := geohash.Encode(lowerLat, lowerLon, returnedAddress.Precision)
	upperHash := geohash.Encode(upperLat, upperLon, returnedAddress.Precision)
	qPNearest := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	qPNearest.Filter("Geohash >=", lowerHash).Filter("Geohash <=", upperHash).Filter("AddressID !=", addressID)
	allNearbyAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	keysClinics, err = db.client.GetAll(ctx, qPNearest, &allNearbyAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return allNearbyAddresses, &currentLatLong, nil
}

// GetNearbySpecialist ....
func (db *dsClinicMeta) GetNearbySpecialist(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string, distance float64) ([]contracts.PhysicalClinicMapLocation, *contracts.ClinicLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	returnedAddress := returnedAddresses[0]
	currentLatLong := returnedAddress.Location
	lat := 0.0144927536231884 // degrees latitude per mile
	lon := 0.0181818181818182 // degrees longitude per mile
	lowerLat := currentLatLong.Lat - lat*distance
	lowerLon := currentLatLong.Long - lon*distance

	upperLat := currentLatLong.Lat + lat*distance
	upperLon := currentLatLong.Long + lon*distance
	lowerHash := geohash.Encode(lowerLat, lowerLon, returnedAddress.Precision)
	upperHash := geohash.Encode(upperLat, upperLon, returnedAddress.Precision)
	qPNearest := datastore.NewQuery("ClinicAddress")
	qPNearest = qPNearest.Filter("Geohash >=", lowerHash).Filter("Geohash <=", upperHash)
	allNearbyAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	keysClinics, err = db.client.GetAll(ctx, qPNearest, &allNearbyAddresses)
	if err != nil {
		return nil, nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return allNearbyAddresses, &currentLatLong, nil
}

// Close closes the database.
func (db *dsClinicMeta) Close() error {
	return db.client.Close()
}
