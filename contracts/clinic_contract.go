package contracts

import "context"

// ClinicRegistrationData ...
// We will secure the connection to backend via SSL/TLS certificates over HTTPS
// So we dont care to send over these details without hashing over the internet
type ClinicRegistrationData struct {
	EmailID    string `json:"emailId" valid:"required"`
	IsVerified bool   `json:"isVerified" valid:"required"`
}

// GetNearbySpecialists .....
type GetNearbySpecialists struct {
	ClinicAddessID string   `json:"clinicAddressId" valid:"required"`
	Specialities   []string `json:"specialities"`
	SearchRadius   string   `json:"searchRadius"`
	Cursor         string   `json:"cursor"`
}

// ClinicVerificationData ...
type ClinicVerificationData struct {
	IsVerified bool `json:"isVerified" valid:"required"`
}

// ClinicRegistrationResponse ....
type ClinicRegistrationResponse struct {
	EmailID    string `json:"emailId" valid:"required"`
	IsVerified bool   `json:"isVerified" valid:"required"`
}

// ClinicRegistrationDatabase provides thread-safe access to a database of UserRegistration.
type ClinicRegistrationDatabase interface {
	//InitializeDataBase initialize computation database
	InitializeDataBase(ctx context.Context, projectID string) error
	// AddClinicRegistration returns a unique user id in datastore that will be used to add others
	AddClinicRegistration(ctx context.Context, clinic *ClinicRegistrationData, uID string) error
	// VerifyUserInDatastore returns a unique user id in datastore that will be used to add others
	VerifyClinicInDatastore(ctx context.Context, emailID string, uID string) error
	// Close closes the database, freeing up any available resources.
	// TODO(cbro): Close() should return an error.
	Close() error
}
