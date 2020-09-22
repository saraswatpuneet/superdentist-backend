package contracts

import (
	"context"

	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"googlemaps.github.io/maps"
)

// PhysicalClinicsRegistration ...
type PhysicalClinicsRegistration struct {
	ClinicAddressID   string         `json:"clinicAddressId"`
	ClinicName        string         `json:"clinicName" valid:"required"`
	ClinicAddress     string         `json:"clinicAddress" valid:"required"`
	ClinicPhoneNumber string         `json:"clinicPhoneNumber" valid:"required"`
	ClinicSpeciality  []string       `json:"clinicSpeciality"`
}

//ClinicAddressResponse ....
type ClinicAddressResponse struct {
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}

// PhysicalClinicMapLocation ....
type PhysicalClinicMapLocation struct {
	PhysicalClinicsRegistration
	Location ClinicLocation
}

//GetClinicAddressResponse ....
type GetClinicAddressResponse struct {
	ClinicDetails []PhysicalClinicMapLocation `json:"clinicDetails" valid:"required"`
	ClinicType    string                        `json:"clinicType" valid:"required"`
}

// ClinicDoctorRegistration ...
type ClinicDoctorRegistration struct {
	ClinicAddressID    string   `json:"clinicAddressId"`
	DoctorPrefix       string   `json:"doctorPrefix" valid:"required"`
	DoctorFirstName    string   `json:"doctorFirstName" valid:"required"`
	DoctorLastName     string   `json:"doctorLastName" valid:"required"`
	DoctorEmailAddress string   `json:"doctorEmailAddress" valid:"required"`
	DoctorSpeciality   []string `json:"doctorSpeciality" valid:"required"`
}

// PostPhysicalClinicDetails .....
type PostPhysicalClinicDetails struct {
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}

// ClinicDoctorsDetails ....
type ClinicDoctorsDetails struct {
	ClinicAddressID string                     `json:"clinicAddressId" valid:"required"`
	Doctors         []ClinicDoctorRegistration `json:"doctors" valid:"required"`
}

//PostDoctorDetails ....
type PostDoctorDetails struct {
	Doctors []ClinicDoctorsDetails `json:"doctorDetails" valid:"required"`
}

// PostPMSDetails .....
type PostPMSDetails struct {
	PMSNames []string `json:"pmsNames" valid:"required"`
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
	// AddServicesForClinic add services offered by clinic
	AddServicesForClinic(ctx context.Context, clinicEmailID string, clinicFBID string, serviceData []ServiceObject) error
	// GetAllClinics get all clinics associated by admin
	GetAllClinics(ctx context.Context, clinicEmailID string, clinicFBID string) ([]PhysicalClinicMapLocation, string, error)
	// GetClinicDoctors ... get doctors either all or for sepecific clinic address
	GetClinicDoctors(ctx context.Context, clinicEmailID string, clinicFBID string, ClinicAddressID string) ([]ClinicDoctorRegistration, error)
	// TODO(cbro): Close() should return an error.
	Close() error
}
