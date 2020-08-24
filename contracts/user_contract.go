package contracts

// UserRegistrationRequest ...
// We will secure the connection to backend via SSL/TLS certificates over HTTPS
// So we dont care to send over these details without hashing over the internet
type UserRegistrationRequest struct {
	FirstName   string `json:"firstName" valid:"required"`
	LastName    string `json:"lastName" valid:"required"`
	Email       string `json:"email" valid:"required"`
	Password    string `json:"password" valid:"required"`
	PhoneNumber string `json:"phoneNumber" valid:"required"`
	// More fields will be added as we progress
}
