package datastoredb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/datastore"
	guuid "github.com/google/uuid"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/geohash"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"
)
// SPECIALITYMAP ....
var SPECIALITYMAP = map[string]string{
	"ortho":  "Orthodontist",
	"maxi":   "Oral and Maxillofacial",
	"oral":   "Oral Surgeon",
	"pedia":  "Perdiatric Dentistry",
	"endo":   "Endodontist",
	"perio":  "Periodontist",
	"prosth": "Prosthodontist",
}

// DSClinicMeta ...
type DSClinicMeta struct {
	projectID string
	client    *datastore.Client
}

//NewClinicMetaHandler return new database action
func NewClinicMetaHandler() *DSClinicMeta {
	return &DSClinicMeta{projectID: "", client: nil}
}

// Ensure dsClinic conforms to the ClinicPhysicalAddressDatabase interface.

var _ contracts.ClinicPhysicalAddressDatabase = &DSClinicMeta{}

// InitializeDataBase ....
func (db *DSClinicMeta) InitializeDataBase(ctx context.Context, projectID string) error {
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

// AddPhysicalAddessressToClinic ...
func (db DSClinicMeta) AddPhysicalAddessressToClinic(ctx context.Context, clinicEmailID string, clinicFBID string, addresses []contracts.PhysicalClinicsRegistration, mapsClient *gmaps.ClientGMaps) ([]contracts.PhysicalClinicsRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	returnedAddress := make([]contracts.PhysicalClinicsRegistration, 0)
	for _, address := range addresses {
		addrID, err := guuid.NewUUID()
		address.AddressID = addrID.String()
		if err != nil {
			return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		splitAddress := strings.Split(address.Address, ",")[0]

		gmapAddress, err := mapsClient.FindPlacesFromText(address.Name + " " + splitAddress)
		location := contracts.ClinicLocation{
			Lat:  0.0,
			Long: 0.0,
		}
		placeID := ""
		phone := ""
		if err == nil && len(gmapAddress.Results) > 0 {
			for idx, gAddress := range gmapAddress.Results {
				splitG := strings.Split(gAddress.FormattedAddress, ",")[0]
				splitA := strings.Split(address.Address, ",")[0]
				if splitG == splitA {
					currentLocation := gmapAddress.Results[idx]
					location.Lat = currentLocation.Geometry.Location.Lat
					location.Long = currentLocation.Geometry.Location.Lng
					placeID = currentLocation.PlaceID
					gPlace, _ := mapsClient.FindPlaceFromID(placeID)
					if gPlace!= nil {
						phone = gPlace.FormattedPhoneNumber
					}
					break
				}
			}
		}
		existingClinic, err := db.GetSingleClinicViaPlace(ctx, placeID)
		if err == nil && existingClinic != nil && existingClinic.AddressID != "" {
			address.AddressID = existingClinic.AddressID
		}
		addressKey := datastore.NameKey("ClinicAddress", address.AddressID, primaryKey)
		if global.Options.DSName != "" {
			addressKey.Namespace = global.Options.DSName
		}
		if existingClinic != nil && existingClinic.AddressID != "" {
			err = db.client.Delete(ctx, addressKey)
		}
		currentHash := geohash.Encode(location.Lat, location.Long, 12)
		currentLocWithMap := contracts.PhysicalClinicMapLocation{
			PhysicalClinicsRegistration: address,
			Location:                    location,
			Geohash:                     currentHash,
			Precision:                   12,
			PlaceID:                     placeID,
		}
		if currentLocWithMap.AutoEmail == "" {
			autoEmail := strings.Replace(address.AddressID, "-", "", -1) + "@clinic.superdentist.io"
			currentLocWithMap.AutoEmail = autoEmail
		}
		currentLocWithMap.PhoneNumber = phone
		_, err = db.client.Put(ctx, addressKey, &currentLocWithMap)
		if err != nil {
			return nil, fmt.Errorf("cannot register clinic with sd: %v", err)
		}
		returnedAddress = append(returnedAddress, address)
	}
	return returnedAddress, nil
}

// UpdateClinicsWithEmail ...
func (db DSClinicMeta) UpdateClinicsWithEmail(ctx context.Context, clinicEmailID string, places []string) error {
	for _, pid := range places {
		currentClinic, key, err := db.GetSingleClinicViaPlaceKey(ctx, pid)
		if err != nil {
			return err
		}
		if currentClinic.EmailAddress != "" {
			return fmt.Errorf("clinic already accounted for")
		}
		currentClinic.EmailAddress = clinicEmailID
		_, err = db.client.Put(ctx, key, currentClinic)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddPhysicalAddessressToClinicNoAdmin ...
func (db DSClinicMeta) AddPhysicalAddessressToClinicNoAdmin(ctx context.Context, placeID string, favs []string, mapsClient *gmaps.ClientGMaps) (contracts.PhysicalClinicMapLocation, bool, error) {
	addrID, err := guuid.NewUUID()
	var addressDB contracts.PhysicalClinicsRegistration
	var currentLocWithMap contracts.PhysicalClinicMapLocation
	existingClinic, err := db.GetSingleClinicViaPlace(ctx, placeID)
	existed := false
	if err == nil && existingClinic != nil && existingClinic.AddressID != "" {
		addressDB = existingClinic.PhysicalClinicsRegistration
		addressDB.Favorites = append(addressDB.Favorites, favs...)
		currentLocWithMap = *existingClinic
		currentLocWithMap.PhysicalClinicsRegistration = addressDB

		if currentLocWithMap.AutoEmail == "" {
			autoEmail := strings.Replace(addressDB.AddressID, "-", "", -1) + "@clinic.superdentist.io"
			currentLocWithMap.AutoEmail = autoEmail
		}
		existed = true
	} else {
		addressDB.AddressID = addrID.String()
		gmapAddress, err := mapsClient.FindPlaceFromID(placeID)
		if err != nil {
			return currentLocWithMap, false, err
		}
		location := contracts.ClinicLocation{
			Lat:  gmapAddress.Geometry.Location.Lat,
			Long: gmapAddress.Geometry.Location.Lng,
		}
		addressDB.Name = gmapAddress.Name
		addressDB.Favorites = favs
		addressDB.Address = gmapAddress.FormattedAddress
		addressDB.Type = "dentist"
		nameLower := strings.ToLower(addressDB.Name)
		for key := range SPECIALITYMAP {
			if strings.Contains(key, nameLower) {
				addressDB.Type = "specialist"
			}
		}
		addressDB.PhoneNumber = gmapAddress.FormattedPhoneNumber
		currentHash := geohash.Encode(location.Lat, location.Long, 12)
		autoEmail := strings.Replace(addressDB.AddressID, "-", "", -1) + "@clinic.superdentist.io"
		currentLocWithMap = contracts.PhysicalClinicMapLocation{
			PhysicalClinicsRegistration: addressDB,
			Location:                    location,
			Geohash:                     currentHash,
			Precision:                   12,
			PlaceID:                     placeID,
			IsVerified:                  true,
			AutoEmail:                   autoEmail,
		}
	}

	addressKey := datastore.NameKey("ClinicAddress", addressDB.AddressID, nil)
	if global.Options.DSName != "" {
		addressKey.Namespace = global.Options.DSName
	}
	if existingClinic != nil && existingClinic.AddressID != "" {
		err = db.client.Delete(ctx, addressKey)
	}

	_, err = db.client.Put(ctx, addressKey, &currentLocWithMap)
	if err != nil {
		return currentLocWithMap, false, fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return currentLocWithMap, existed, nil
}

// AddClinicJoinURL ....
func (db DSClinicMeta) AddClinicJoinURL(ctx context.Context, currentClinic contracts.PhysicalClinicMapLocation, url string) {
	var joinDetails contracts.ClinicJoinDetails
	joinDetails.Name = currentClinic.Name
	joinDetails.Address = currentClinic.Address
	joinDetails.URL = url
	joinDetails.PlaceID = currentClinic.PlaceID
	joinDetails.AutoEmail = currentClinic.AutoEmail
	numKey := datastore.NameKey("ClinicJoinDetails", currentClinic.PlaceID, nil)
	if global.Options.DSName != "" {
		numKey.Namespace = global.Options.DSName
	}
	db.client.Put(ctx, numKey, &joinDetails)
}

// DeleteClinicJoinURL ....
func (db DSClinicMeta) DeleteClinicJoinURL(ctx context.Context, places []string) {
	for _, pid := range places {
		qP := datastore.NewQuery("ClinicJoinDetails")
		qP = qP.Filter("PlaceID =", pid)
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		numKey := datastore.NameKey("ClinicJoinDetails", pid, nil)
		if global.Options.DSName != "" {
			numKey.Namespace = global.Options.DSName
		}
		db.client.Delete(ctx, numKey)
	}
}

// UpdatePhysicalAddessressToClinic ....
func (db DSClinicMeta) UpdatePhysicalAddessressToClinic(ctx context.Context, clinicFBID string, clinicUpdated contracts.PhysicalClinicMapLocation) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicUpdated.EmailAddress, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	addressKey := datastore.NameKey("ClinicAddress", clinicUpdated.AddressID, primaryKey)
	if global.Options.DSName != "" {
		addressKey.Namespace = global.Options.DSName
	}
	_, err := db.client.Put(ctx, addressKey, &clinicUpdated)
	if err != nil {
		return fmt.Errorf("update clinic failed: %v", err)
	}
	return nil
}

// UpdateNetworkForFavoritedClinic .....
func (db DSClinicMeta) UpdateNetworkForFavoritedClinic(ctx context.Context, clinicUpdated contracts.PhysicalClinicMapLocation) error {
	for _, favClinic := range clinicUpdated.Favorites {
		primaryKey := datastore.NameKey("ClinicNetwork", favClinic, nil)
		if global.Options.DSName != "" {
			primaryKey.Namespace = global.Options.DSName
		}
		var clinicNetwork contracts.ClinicNetwork
		err := db.client.Get(ctx, primaryKey, clinicNetwork)
		if err != nil || clinicNetwork.ClinicPlaceID == nil || len(clinicNetwork.ClinicPlaceID) <= 0 {
			networkData := []string{clinicUpdated.PlaceID}
			clinicNetwork.ClinicPlaceID = networkData
		} else {
			clinicNetwork.ClinicPlaceID = append(clinicNetwork.ClinicPlaceID, clinicUpdated.PlaceID)
		}
		_, err = db.client.Put(ctx, primaryKey, &clinicNetwork)
		if err != nil {
			return fmt.Errorf("cannot update clinic network: %v", err)
		}
	}
	return nil
	//lets create the clinic
}

// RemoveNetworkForFavoritedClinic ...
func (db DSClinicMeta) RemoveNetworkForFavoritedClinic(ctx context.Context, favID string, favClinic string) error {
	primaryKey := datastore.NameKey("ClinicNetwork", favID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	var clinicNetwork contracts.ClinicNetwork
	err := db.client.Get(ctx, primaryKey, clinicNetwork)
	if err != nil || clinicNetwork.ClinicPlaceID == nil || len(clinicNetwork.ClinicPlaceID) <= 0 {
		networkData := []string{}
		clinicNetwork.ClinicPlaceID = networkData
	} else {
		newNetwork := make([]string, 0)
		for _, placeID := range clinicNetwork.ClinicPlaceID {
			if placeID != favClinic {
				newNetwork = append(newNetwork, placeID)
			}
		}
		clinicNetwork.ClinicPlaceID = newNetwork
	}
	_, err = db.client.Put(ctx, primaryKey, &clinicNetwork)
	if err != nil {
		return fmt.Errorf("cannot update clinic network: %v", err)

	}
	return nil
	//lets create the clinic
}

// GetNetworkClincs ....
func (db DSClinicMeta) GetNetworkClincs(ctx context.Context, placeID string) ([]string, error) {
	primaryKey := datastore.NameKey("ClinicNetwork", placeID, nil)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	var clinicNetwork contracts.ClinicNetwork
	err := db.client.Get(ctx, primaryKey, &clinicNetwork)
	if err != nil {
		return []string{}, fmt.Errorf("cannot update clinic network: %v", err)
	}
	return clinicNetwork.ClinicPlaceID, nil
	//lets create the clinic
}

// AddDoctorsToPhysicalClincs ....
func (db DSClinicMeta) AddDoctorsToPhysicalClincs(ctx context.Context, clinicEmailID string, clinicFBID string, doctorsData []contracts.ClinicDoctorsDetails) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	for _, doctor := range doctorsData {
		for _, doc := range doctor.Doctors {
			docID, err := guuid.NewUUID()
			doc.AddressID = doctor.AddressID
			clinicDoctorKey := datastore.NameKey("ClinicDoctors", docID.String(), primaryKey)
			if global.Options.DSName != "" {
				clinicDoctorKey.Namespace = global.Options.DSName
			}
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
func (db *DSClinicMeta) AddPMSUsedByClinics(ctx context.Context, clinicEmailID string, clinicFBID string, pmsData []string) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	clinicPMSKey := datastore.IncompleteKey("ClinicPMS", primaryKey)
	if global.Options.DSName != "" {
		clinicPMSKey.Namespace = global.Options.DSName
	}
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
func (db *DSClinicMeta) AddPMSAuthDetails(ctx context.Context, clinicEmailID string, clinicFBID string, pmsInformation contracts.PostPMSAuthDetails) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	for _, pmsData := range pmsInformation.PMSAuthData {
		clinicPMSKey := datastore.NameKey("ClinicPMSAuth", pmsData.PMSName, primaryKey)
		if global.Options.DSName != "" {
			clinicPMSKey.Namespace = global.Options.DSName
		}
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
func (db *DSClinicMeta) AddServicesForClinic(ctx context.Context, clinicEmailID string, clinicFBID string, serviceData []contracts.ServiceObject) error {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	for _, serObj := range serviceData {
		clinicPMSKey := datastore.IncompleteKey("ClinicServices", primaryKey)
		if global.Options.DSName != "" {
			clinicPMSKey.Namespace = global.Options.DSName
		}
		_, err := db.client.Put(ctx, clinicPMSKey, &serObj)
		if err != nil {
			return fmt.Errorf("cannot register clinic with sd: %v", err)
		}
	}

	return nil
}

// StorePNGInDatabase .....
func (db *DSClinicMeta) StorePNGInDatabase(ctx context.Context, png string,
	gdClincs map[string][]contracts.PhysicalClinicMapLocation,
	spClinics map[string][]contracts.PhysicalClinicMapLocation) error {
	qrID, _ := guuid.NewUUID()
	qrTextID := qrID.String()
	var storeQR contracts.QRStoreSchema
	parentKey := datastore.NameKey("ClinicQR", qrTextID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	foundExisting := false
	qrExists := ""
	var qrKey *datastore.Key
	for _, values := range gdClincs {
		for _, cli1 := range values {
			for _, values := range spClinics {
				for _, cli2 := range values {
					existingQR, key, err := db.GetStoreKeysQR(ctx, cli1.PlaceID, cli2.PlaceID)
					if err != nil {
						continue
					} else if err == nil && existingQR != "" {
						foundExisting = true
						qrExists = existingQR
						qrKey = key
						break
					}
				}
				if foundExisting {
					break
				}
			}
			if foundExisting {
				break
			}
		}
		if foundExisting {
			break
		}
	}
	for _, values := range gdClincs {
		for _, cli := range values {
			storeQR.GDID = append(storeQR.SPID, cli.PlaceID)
		}
	}
	for _, values := range spClinics {
		for _, cli := range values {
			storeQR.SPID = append(storeQR.SPID, cli.PlaceID)
		}
	}
	if foundExisting && qrExists != "" && qrKey != nil {
		parentKey = qrKey
	}
	storeQR.QRCode = png
	_, err := db.client.Put(ctx, parentKey, &storeQR)
	if err != nil {
		return fmt.Errorf("cannot register clinic with sd: %v", err)
	}
	return nil
}

// GetQRFROMDatabase ....
func (db *DSClinicMeta) GetQRFROMDatabase(ctx context.Context,
	gdPlaceID string,
	spPlaceID string) (string, error) {
	returnedAddresses := make([]contracts.QRStoreSchema, 0)

	qP := datastore.NewQuery("ClinicQR")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	qP = qP.Filter("GDID =", gdPlaceID).Filter("SPID =", spPlaceID)
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return "", fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return returnedAddresses[0].QRCode, nil
}

// GetStoreKeysQR ....
func (db *DSClinicMeta) GetStoreKeysQR(ctx context.Context,
	gdPlaceID string,
	spPlaceID string) (string, *datastore.Key, error) {
	returnedAddresses := make([]contracts.QRStoreSchema, 0)

	qP := datastore.NewQuery("ClinicQR")
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	qP = qP.Filter("GDID =", gdPlaceID).Filter("SPID =", spPlaceID)
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return "", nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return returnedAddresses[0].QRCode, keysClinics[0], nil
}

// GetAllClinics ....
func (db *DSClinicMeta) GetAllClinics(ctx context.Context, clinicEmailID string, clinicFBID string) ([]contracts.PhysicalClinicMapLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	returnedAddress := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddress)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return returnedAddress, nil
}

// GetAllClinicsByEmail ....
func (db *DSClinicMeta) GetAllClinicsByEmail(ctx context.Context, clinicEmailID string) ([]contracts.PhysicalClinicMapLocation, error) {
	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress")
	if clinicEmailID != "" {
		qP = qP.Filter("EmailAddress =", clinicEmailID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return returnedAddresses, nil
}

// GetClinicDoctors ....
func (db *DSClinicMeta) GetClinicDoctors(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string) ([]contracts.ClinicDoctorRegistration, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	returnedDoctors := make([]contracts.ClinicDoctorRegistration, 0)
	qP := datastore.NewQuery("ClinicDoctors").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedDoctors)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	return returnedDoctors, nil
}

// GetSingleClinic ....
func (db *DSClinicMeta) GetSingleClinic(ctx context.Context, addressID string) (*contracts.PhysicalClinicMapLocation, error) {

	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress")
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return &returnedAddresses[0], nil
}

// GetSingleClinicViaPlace ....
func (db *DSClinicMeta) GetSingleClinicViaPlace(ctx context.Context, placeID string) (*contracts.PhysicalClinicMapLocation, error) {

	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress")
	if placeID != "" {
		qP = qP.Filter("PlaceID =", placeID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return &returnedAddresses[0], nil
}

// GetSingleClinicViaPlaceKey ....
func (db *DSClinicMeta) GetSingleClinicViaPlaceKey(ctx context.Context, placeID string) (*contracts.PhysicalClinicMapLocation, *datastore.Key, error) {

	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress")
	if placeID != "" {
		qP = qP.Filter("PlaceID =", placeID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, nil, fmt.Errorf("clinic with given address id not found: %v", err)
	}
	return &returnedAddresses[0], keysClinics[0], nil
}

// GetNearbyClinics ....
func (db *DSClinicMeta) GetNearbyClinics(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string, distance float64) ([]contracts.PhysicalClinicMapLocation, *contracts.ClinicLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
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
	if global.Options.DSName != "" {
		qPNearest = qPNearest.Namespace(global.Options.DSName)
	}
	keysClinics, err = db.client.GetAll(ctx, qPNearest, &allNearbyAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return allNearbyAddresses, &currentLatLong, nil
}

// GetNearbySpecialist ....
func (db *DSClinicMeta) GetNearbySpecialist(ctx context.Context, clinicEmailID string, clinicFBID string, addressID string, distance float64) ([]contracts.PhysicalClinicMapLocation, error) {
	parentKey := datastore.NameKey("ClinicAdmin", clinicFBID, nil)
	if global.Options.DSName != "" {
		parentKey.Namespace = global.Options.DSName
	}
	primaryKey := datastore.NameKey("ClinicAdmin", clinicEmailID, parentKey)
	if global.Options.DSName != "" {
		primaryKey.Namespace = global.Options.DSName
	}
	returnedAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	qP := datastore.NewQuery("ClinicAddress").Ancestor(primaryKey)
	if addressID != "" {
		qP = qP.Filter("AddressID =", addressID)
	}
	if global.Options.DSName != "" {
		qP = qP.Namespace(global.Options.DSName)
	}
	keysClinics, err := db.client.GetAll(ctx, qP, &returnedAddresses)
	if err != nil || len(keysClinics) <= 0 {
		return nil, fmt.Errorf("no doctors have been found for the given clinic address: %v", err)
	}
	returnedAddress := returnedAddresses[0]
	currentHash := returnedAddress.Geohash
	lowerHash := string(currentHash[0:4])
	upperHash := lowerHash + "~"
	qPNearest := datastore.NewQuery("ClinicAddress")
	qPNearest = qPNearest.Filter("Geohash >=", lowerHash).Filter("Geohash <=", upperHash)
	allNearbyAddresses := make([]contracts.PhysicalClinicMapLocation, 0)
	if global.Options.DSName != "" {
		qPNearest = qPNearest.Namespace(global.Options.DSName)
	}
	keysClinics, err = db.client.GetAll(ctx, qPNearest, &allNearbyAddresses)
	if err != nil {
		return nil, fmt.Errorf("no clinics have been found for the admin error: %v", err)
	}
	return allNearbyAddresses, nil
}

// GetFavoriteSpecialists ....
func (db *DSClinicMeta) GetFavoriteSpecialists(ctx context.Context, clinicEmailID string, clinicFBID string, currentFavorites []string) ([]contracts.PhysicalClinicMapLocation, error) {

	allNearbyAddresses := make([]contracts.PhysicalClinicMapLocation, 0)

	for _, placeID := range currentFavorites {
		var currentVerified contracts.PhysicalClinicMapLocation
		tempFav := make([]contracts.PhysicalClinicMapLocation, 0)

		qP := datastore.NewQuery("ClinicAddress").Filter("PlaceID =", placeID)
		if global.Options.DSName != "" {
			qP = qP.Namespace(global.Options.DSName)
		}
		_, err := db.client.GetAll(ctx, qP, &tempFav)
		if err != nil || len(tempFav) == 0 {
			currentVerified.PlaceID = placeID
			currentVerified.IsVerified = false
			allNearbyAddresses = append(allNearbyAddresses, currentVerified)
			continue
		}
		currentVerified = tempFav[0]
		currentVerified.IsVerified = true
		allNearbyAddresses = append(allNearbyAddresses, currentVerified)
	}
	return allNearbyAddresses, nil
}

// Close closes the database.
func (db *DSClinicMeta) Close() error {
	return db.client.Close()
}
