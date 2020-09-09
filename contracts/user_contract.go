package contracts

// UserRegistrationRequest ...
// We will secure the connection to backend via SSL/TLS certificates over HTTPS
// So we dont care to send over these details without hashing over the internet
type UserRegistrationRequest struct {
	EmailID       string `json:"emailId" valid:"required"`
	UserRole 	string `json:"userRole" valid:"required"`
	PhoneNumber string `json:"phoneNumber"`
}
