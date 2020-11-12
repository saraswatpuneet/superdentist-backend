package contracts

// Patient ....
type Patient struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Dob       string `json:"dob"`
	Email     string `json:"email"`
	MemberID  string `json:"member_id"`
	GroupID   string `json:"group_number"`
}

// Payer ....
type Payer struct {
	ID   string `json:"id" valid:"required"`
	Name string `json:"name" valid:"required"`
}
