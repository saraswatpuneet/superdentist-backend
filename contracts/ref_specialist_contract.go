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

// Comment .....
type Comment struct {
	Time      string `json:"time" valid:"required"`
	Comment   string `json:"comment" valid:"required"`
	CreatedBy string `json:"createdBy" valid:"required"`
	ChatBox   string `json:"chatBox" valid:"required"`
}

// Status ....
type Status struct {
	GDStatus string `json:"gdStatus" valid:"required"`
	SPStatus string `json:"spStatus" valid:"required"`
}

// ReferralDetails ....
type ReferralDetails struct {
	Patient       Patient   `json:"patient" valid:"required"`
	FromAddressID string    `json:"fromAddressId" valid:"required"`
	ToAddressID   string    `json:"toAddressId" valid:"required"`
	ToPlaceID     string    `json:"toPlaceId"`
	Status        Status    `json:"status" valid:"required"`
	Comments      []Comment `json:"comments"`
	Reasons       []string  `json:"reasons"`
	History       []string  `json:"history"`
	Tooth         []string  `json:"tooth"`
}

// ReferralComments .....
type ReferralComments struct {
	Comments []Comment `json:"comments"`
}

// ReferralStatus .....
type ReferralStatus struct {
	Status Status `json:"status"`
}

// DSReferral .....
type DSReferral struct {
	ReferralID        string    `json:"referralId" valid:"required"`
	Documents         []string  `json:"documents" valid:"required"`
	FromPlaceID       string    `json:"fromPlaceID" valid:"required"`
	ToPlaceID         string    `json:"toPlaceID" valid:"required"`
	FromClinicName    string    `json:"fromClinicName" valid:"required"`
	ToClinicName      string    `json:"toClinicName" valid:"required"`
	FromClinicAddress string    `json:"fromClinicAddress" valid:"required"`
	ToClinicAddress   string    `json:"toClinicAddress" valid:"required"`
	FromAddressID     string    `json:"fromAddressId" valid:"required"`
	ToAddressID       string    `json:"toAddressId" valid:"required"`
	Comments          []Comment `json:"comments"`
	Status            Status    `json:"status" valid:"required"`
	Reasons           []string  `json:"reasons"`
	History           []string  `json:"history"`
	Tooth             []string  `json:"tooth"`
	CreatedOn         time.Time `json:"createdOn" valid:"required"`
	ModifiedOn        time.Time `json:"modifiedOn" valid:"required"`
	PatientEmail      string    `json:"patientEmail" valid:"required"`
	PatientFirstName  string    `json:"patientFirstName" valid:"required"`
	PatientLastName   string    `json:"patientLastName" valid:"required"`
	PatientPhone      string    `json:"patientPhone" valid:"required"`
	FromEmail         string    `json:"fromEmail" valid:"required"`
	ToEmail           string    `json:"toEmail" valid:"required"`
	IsDirty           bool      `json:"isDirty" valid:"required"`
}
