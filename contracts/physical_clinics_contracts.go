package contracts

import (
	"context"

	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"googlemaps.github.io/maps"
)

// PhysicalClinicsRegistration ...
type PhysicalClinicsRegistration struct {
	Type         string   `json:"type" valid:"required"`
	AddressID    string   `json:"addressId"`
	Name         string   `json:"name" valid:"required"`
	Address      string   `json:"address" valid:"required"`
	EmailAddress string   `json:"emailAddress" valid:"required"`
	PhoneNumber  string   `json:"phoneNumber" valid:"required"`
	Specialty    []string `json:"specialty"`
	Favorites    []string `json:"favorites"`
}

//ClinicAddressResponse ....
type ClinicAddressResponse struct {
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}

// PhysicalClinicMapLocation ....
type PhysicalClinicMapLocation struct {
	PhysicalClinicsRegistration
	Location   ClinicLocation
	IsVerified bool
	Geohash    string `json:"-"`
	Precision  int    `json:"-"`
	PlaceID    string
}

// QRStoreSchema ....
type QRStoreSchema struct {
	GDID   []string
	SPID   []string
	QRCode string
}

// PhysicalClinicMapDetails ....
type PhysicalClinicMapDetails struct {
	VerifiedDetails PhysicalClinicMapLocation `json:"verifiedDetails" valid:"required"`
	GeneralDetails  maps.PlaceDetailsResult   `json:"generalDetails" valid:"required"`
	QRCode          string                    `json:"qrCode"`
}

//GetClinicAddressResponse ....
type GetClinicAddressResponse struct {
	ClinicDetails []PhysicalClinicMapLocation `json:"clinicDetails" valid:"required"`
}

// GetNearbyClinics ....
type GetNearbyClinics struct {
	ClinicAddresses []PhysicalClinicMapDetails `json:"clinicAddresses" valid:"required"`
	Cursor          string                     `json:"cursor"`
}

// GetFavClinics ....
type GetFavClinics struct {
	ClinicAddresses []PhysicalClinicMapDetails `json:"clinicAddresses" valid:"required"`
}

// ClinicDoctorRegistration ...
type ClinicDoctorRegistration struct {
	AddressID    string   `json:"addressId"`
	Prefix       string   `json:"prefix" valid:"required"`
	FirstName    string   `json:"firstName" valid:"required"`
	LastName     string   `json:"lastName" valid:"required"`
	EmailAddress string   `json:"emailAddress" valid:"required"`
	Specialty    []string `json:"specialty" valid:"required"`
}

// PostPhysicalClinicDetails .....
type PostPhysicalClinicDetails struct {
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}

// ClinicDoctorsDetails ....
type ClinicDoctorsDetails struct {
	AddressID string                     `json:"addressId" valid:"required"`
	Doctors   []ClinicDoctorRegistration `json:"doctors" valid:"required"`
}

//PostDoctorDetails ....
type PostDoctorDetails struct {
	Doctors []ClinicDoctorsDetails `json:"doctorDetails" valid:"required"`
}

// PostPMSDetails .....
type PostPMSDetails struct {
	PMSNames []string `json:"pmsNames" valid:"required"`
}

// ClinicNetwork ....
type ClinicNetwork struct {
	ClinicPlaceID []string `json:"clinicPlaceID" valid:"required"`
}

// PMSAuthStruct .....
type PMSAuthStruct struct {
	PMSName     string                 `json:"pmsName" valid:"required"`
	AuthDetails map[string]interface{} `json:"authDetails" valid:"required"`
}

// PMSAuthStructStore .....
type PMSAuthStructStore struct {
	PMSName     string `json:"pmsName" valid:"required"`
	AuthDetails string `json:"authDetails" valid:"required"`
}

// PostPMSAuthDetails ..
type PostPMSAuthDetails struct {
	PMSAuthData []PMSAuthStruct `json:"pmsAuthData" valid:"required"`
}

//ServiceObject .....
type ServiceObject struct {
	ServiceGroup string   `json:"serviceGroup" valid:"required"`
	ServiceList  []string `json:"serviceList" valid:"required"`
}

// PostAddressList ....
type PostAddressList struct {
	AddressList []maps.PlacesSearchResult `json:"addressList" valid:"required"`
	Error       string                    `json:"error" valid:"required"`
}

// PostClinicServices ....
type PostClinicServices struct {
	Services []ServiceObject `json:"services" valid:"required"`
}

// ClinicLocation .....
type ClinicLocation struct {
	Lat  float64 `json:"lat" valid:"required"`
	Long float64 `json:"long" valid:"required"`
}

// ClinicPhysicalAddressDatabase provides thread-safe access to a database of UserRegistration.
type ClinicPhysicalAddressDatabase interface {
	//InitializeDataBase initialize computation database
	InitializeDataBase(ctx context.Context, projectID string) error
	// AddPhysicalAddessressToClinic .......
	AddPhysicalAddessressToClinic(ctx context.Context, clinicEmailID string, clinicFBID string, addresses []PhysicalClinicsRegistration, mapsClient *gmaps.ClientGMaps) ([]PhysicalClinicsRegistration, error)
	// AddDoctorsToPhysicalClincs ....
	AddDoctorsToPhysicalClincs(ctx context.Context, clinicEmailID string, clinicFBID string, doctorsData []ClinicDoctorsDetails) error // Close closes the database, freeing up any available resources.
	// AddPMSUsedByClinics PMS to DB
	AddPMSUsedByClinics(ctx context.Context, clinicEmailID string, clinicFBID string, pmsData []string) error
	// AddPMSAuthDetails PMS Auth to DB
	AddPMSAuthDetails(ctx context.Context, clinicEmailID string, clinicFBID string, pmsData PostPMSAuthDetails) error
	// AddServicesForClinic add services offered by clinic
	AddServicesForClinic(ctx context.Context, clinicEmailID string, clinicFBID string, serviceData []ServiceObject) error
	// GetAllClinics get all clinics associated by admin
	GetAllClinics(ctx context.Context, clinicEmailID string, clinicFBID string) ([]PhysicalClinicMapLocation, error)
	// GetClinicDoctors ... get doctors either all or for sepecific clinic address
	GetClinicDoctors(ctx context.Context, clinicEmailID string, clinicFBID string, AddressID string) ([]ClinicDoctorRegistration, error)
	// TODO(cbro): Close() should return an error.
	Close() error
}
