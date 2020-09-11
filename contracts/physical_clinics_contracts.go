package contracts

// PhysicalClinicsRegistration ...
type PhysicalClinicsRegistration struct {
	ClinicName        string `json:"clinicName" valid:"required"`
	ClinicAddress     string `json:"clinicAddress" valid:"required"`
	ClinicPhoneNumber string `json:"clinicPhoneNumber" valid:"required"`
}

// PostPhysicalClinicDetails .....
type PostPhysicalClinicDetails struct {
	EmailID       string                        `json:"emailId" valid:"required"`
	ClinicID      string                        `json:"clinicId" valid:"required"`
	ClinicDetails []PhysicalClinicsRegistration `json:"clinicDetails" valid:"required"`
}