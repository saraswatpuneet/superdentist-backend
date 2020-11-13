package contracts

import "time"

// Patient ....
type Patient struct {
	FirstName string `json:"first_name" valid:"required"`
	LastName  string `json:"last_name" valid:"required"`
	Dob       string `json:"dob"`
	Email     string `json:"email" valid:"required"`
	Phone     string `json:"phone" valid:"required"`
	MemberID  string `json:"member_id"`
	GroupID   string `json:"group_number"`
}

// ReferralDetails ....
type ReferralDetails struct {
	Patient       Patient `json:"patient" valid:"required"`
	FromAddressID string  `json:"fromAddressId" valid:"required"`
	ToAddressID   string  `json:"toAddressId" valid:"required"`
	ToPlaceID     string  `json:"toPlaceId"`
	Status        string  `json:"status" valid:"required"`
	Comments      string  `json:"comments"`
}

// DSReferral .....
type DSReferral struct {
	ReferralDetails
	ReferralID string    `json:"referralId" valid:"required"`
	Documents  []string  `json:"documents" valid:"required"`
	CreatedOn  time.Time `json:"createdOn" valid:"required"`
	ModifiedOn time.Time `json:"modifiedOn" valid:"required"`
}
