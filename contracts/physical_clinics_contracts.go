package contracts

import "context"

// PhysicalClinicsRegistration ...
type PhysicalClinicsRegistration struct {
	ClinicAddressID   string `json:"clinicAddressId"`
	ClinicName        string `json:"clinicName" valid:"required"`
	ClinicAddress     string `json:"clinicAddress" valid:"required"`
	ClinicPhoneNumber string `json:"clinicPhoneNumber" valid:"required"`
	ClinicSpeciality   []string `json:"clinicSpeciality"`
}

//ClinicAddressResponse ....
type ClinicAddressResponse struct {
	ClinicID      string                        `json:"clinicId" valid:"required"`
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}

// ClinicDoctorRegistration ...
type ClinicDoctorRegistration struct {
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
	PMSNames [] string `json:"pmsNames" valid:"required"`
}
// ClinicPhysicalAddressDatabase provides thread-safe access to a database of UserRegistration.
type ClinicPhysicalAddressDatabase interface {
	//InitializeDataBase initialize computation database
	InitializeDataBase(ctx context.Context, projectID string) error
	// AddPhysicalAddessressToClinic .......
	AddPhysicalAddessressToClinic(ctx context.Context, clinicEmailID string, clinicFBID string, addresses []PhysicalClinicsRegistration) ([]PhysicalClinicsRegistration, error)
	// AddDoctorsToPhysicalClincs ....
	AddDoctorsToPhysicalClincs(ctx context.Context, clinicEmailID string, clinicFBID string, doctorsData []ClinicDoctorsDetails) error // Close closes the database, freeing up any available resources.
	// Add PMS to DB 
	AddPMSUsedByClinics(ctx context.Context,  clinicEmailID string, clinicFBID string, pmsData []string) error
	// TODO(cbro): Close() should return an error.
	Close() error
}
