package contracts

import (
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// DOB ....
type DOB struct {
	Year  string `json:"year" valid:"required"`
	Month string `json:"month" valid:"required"`
	Day   string `json:"day" valid:"required"`
}

// Subscriber ...
type Subscriber struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	DOB       DOB    `json:"dob"`
}

// PatientDentalInsurance ....
type PatientDentalInsurance struct {
	Company    string        `json:"company"`
	MemberID   string        `json:"memberId"`
	Subscriber Subscriber    `json:"subscriber"`
	CompanyID  string        `json:"companyId"`
	Status     PatientStatus `json:"status"`
	ID         string        `json:"id"`
	AgentID    string        `json:"agentId"`
	PatientID  string        `json:"patientId"`
	AddressID  string        `json:"addressId"`
	DueDate    int64         `json:"dueDate"`
}

// PatientMedicalInsurance ....
type PatientMedicalInsurance struct {
	Company     string        `json:"company"`
	GroupNumber string        `json:"groupNumber"`
	MemberID    string        `json:"memberId"`
	Subscriber  Subscriber    `json:"subscriber"`
	SSN         string        `json:"ssn"`
	Status      PatientStatus `json:"status"`
	ID          string        `json:"id"`
	AgentID     string        `json:"agentId"`
	PatientID   string        `json:"patientId"`
	AddressID   string        `json:"addressId"`
	DueDate     int64         `json:"dueDate"`
}

// notes: clinic info, tax id, group npi, provider name, provider npi,

// PatientStatus ...
type PatientStatus struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// PatientInsuranceAgent ....

type PatientInsuranceAgent struct {
	PatientID string                  `json:"patientId" valid:"required"`
	AgentID   string                  `json:"agentId" valid:"required"`
	Dental    PatientDentalInsurance  `json:"dental"`
	Medical   PatientMedicalInsurance `json:"medical"`
}

// Patient ....
type Patient struct {
	PatientID        string                    `json:"patientId" valid:"required"`
	FirstName        string                    `json:"firstName" valid:"required"`
	LastName         string                    `json:"lastName" valid:"required"`
	Dob              DOB                       `json:"dob"`
	Email            string                    `json:"email" valid:"required"`
	Phone            string                    `json:"phone" valid:"required"`
	SSN              string                    `json:"_"`
	DentalInsurance  []PatientDentalInsurance  `json:"dentalInsurance" datastore:"dentalInsurance,noindex"`
	MedicalInsurance []PatientMedicalInsurance `json:"medicalInsurance" datastore:"medicalInsurance,noindex"`
	GDName           string                    `json:"gdName" valid:"required"`
	SP               string                    `json:"sp" valid:"required"`
	GD               string                    `json:"gd" valid:"required"`
	AddressID        string                    `json:"addressId" valid:"required"`
	ClinicName       string                    `json:"clinicName" valid:"required"`
	SPName           string                    `json:"spName" valid:"required"`
	ReferralID       string                    `json:"referralId" valid:"required"`
	DueDate          int64                     `json:"dueDate"`
	AppointmentTime  string                    `json:"appointmentTime"`
	SameDay          bool                      `json:"sameDay"`
	Status           PatientStatus             `json:"status"`
	ZipCode          string                    `json:"zipCode"`
	CreatedOn        int64                     `json:"createdOn"`
	CreationDate     string                    `json:"creationDate"`
	VisitCount       int                       `json:"visitCount"`
	LastAppointment  int64                     `json:"lastAppointment"`
}

// PatientStore ....
type PatientStore struct {
	PatientID          string        `json:"patientId" valid:"required"`
	FirstName          string        `json:"firstName" valid:"required"`
	LastName           string        `json:"lastName" valid:"required"`
	Dob                DOB           `json:"dob"`
	Email              string        `json:"email" valid:"required"`
	Phone              string        `json:"phone" valid:"required"`
	SSN                string        `json:"_"`
	DentalInsuraceID   []string      `json:"dentalInsuranceId"`
	MedicalInsuranceID []string      `json:"medicalInsuranceId"`
	GDName             string        `json:"gdName" valid:"required"`
	SP                 string        `json:"sp" valid:"required"`
	GD                 string        `json:"gd" valid:"required"`
	AddressID          string        `json:"addressId" valid:"required"`
	ClinicName         string        `json:"clinicName" valid:"required"`
	SPName             string        `json:"spName" valid:"required"`
	ReferralID         string        `json:"referralId" valid:"required"`
	DueDate            int64         `json:"dueDate"`
	LastAppointment    int64         `json:"lastAppointment"`
	VisitCount         int           `json:"visitCount"`
	AppointmentTime    string        `json:"appointmentTime"`
	SameDay            bool          `json:"sameDay"`
	Status             PatientStatus `json:"status"`
	ZipCode            string        `json:"zipCode"`
	CreatedOn          int64         `json:"createdOn"`
	CreationDate       string        `json:"creationDate"`
}

//PatientVerificationStatistics ...
type PatientVerificationStatistics struct {
	TotalPatients    int            `json:"totalPatients"`
	StatusCounts     map[string]int `json:"statusCounts"`
	VisitationCounts map[string]int `json:"visitationCounts"`
}

// AgentInsuranceMap ...
type AgentInsuranceMap struct {
	AgentID     string `json:"agentId"`
	InsuranceID string `json:"insuranceId"`
}

// PatientFilters ...

type PatientFilters struct {
	StartTime int64
	EndTime   int64
	AgentID   string
	Status    string
	Companies []string
}

// PatientList ....
type PatientList struct {
	Patients   []Patient `json:"patients"`
	CursorNext string    `json:"cursorNext"`
}

// SelectedDentalCodes ....
type SelectedDentalCodes struct {
	GroupID string   `json:"groupId"`
	CodeIds []string `json:"codeIds"`
}

// Notes ...
type Notes struct {
	PatientID string `json:"patientId"`
	Type      string
	Details   string `json:"details" datastore:",noindex"`
}

type ClinicSpecificCodes struct {
	PracticeCodes []SelectedDentalCodes `json:"practiceCodes"`
}

// ChatBox ....
type ChatBox string

// GDCBox ....
const GDCBox ChatBox = "c2c"

// SPCBox ....
const SPCBox ChatBox = "c2p"

// Thumbnails
type Media struct {
	Name  string `json:"name"`
	Image string `json:"image" datastore:",noindex"`
}

// Comment .....
type Comment struct {
	MessageID string   `json:"messageId"`
	TimeStamp int64    `json:"timeStamp"`
	Text      string   `json:"text" valid:"required"`
	Channel   ChatBox  `json:"channel" valid:"required"`
	UserID    string   `json:"userId" valid:"required"`
	Files     []string `json:"file"`
	Media     []Media  `json:"media"`
}

// Status ....
type Status struct {
	GDStatus string `json:"gdStatus" valid:"required"`
	SPStatus string `json:"spStatus" valid:"required"`
}

// ReferralDetails ....
type ReferralDetails struct {
	Patient       PatientStore `json:"patient" valid:"required"`
	FromAddressID string       `json:"fromAddressId" valid:"required"`
	ToAddressID   string       `json:"toAddressId" valid:"required"`
	FromPlaceID   string       `json:"fromPlaceId"`
	ToPlaceID     string       `json:"toPlaceId"`
	Status        Status       `json:"status" valid:"required"`
	Comments      []Comment    `json:"comments"`
	Reasons       []string     `json:"reasons"`
	History       []string     `json:"history"`
	Tooth         []string     `json:"tooth"`
	IsSummary     bool         `json:"isSummary"`
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
	ReferralID         string    `json:"referralId" valid:"required"`
	Documents          []string  `json:"documents" valid:"required"`
	FromPlaceID        string    `json:"fromPlaceID" valid:"required"`
	ToPlaceID          string    `json:"toPlaceID" valid:"required"`
	FromClinicName     string    `json:"fromClinicName" valid:"required"`
	ToClinicName       string    `json:"toClinicName" valid:"required"`
	FromClinicAddress  string    `json:"fromClinicAddress" valid:"required"`
	ToClinicAddress    string    `json:"toClinicAddress" valid:"required"`
	FromAddressID      string    `json:"fromAddressId" valid:"required"`
	ToAddressID        string    `json:"toAddressId" valid:"required"`
	Status             Status    `json:"status" valid:"required"`
	Reasons            []string  `json:"reasons"`
	History            []string  `json:"history"`
	Tooth              []string  `json:"tooth"`
	CreatedOn          time.Time `json:"createdOn" valid:"required"`
	ModifiedOn         time.Time `json:"modifiedOn" valid:"required"`
	PatientEmail       string    `json:"patientEmail" valid:"required"`
	PatientFirstName   string    `json:"patientFirstName" valid:"required"`
	PatientLastName    string    `json:"patientLastName" valid:"required"`
	PatientDOBYear     string    `json:"patientDobYear" valid:"required"`
	PatientDOBMonth    string    `json:"patientDobMonth" valid:"required"`
	PatientDOBDay      string    `json:"patientDobDay" valid:"required"`
	PatientPhone       string    `json:"patientPhone" valid:"required"`
	FromClinicPhone    string    `json:"fromClinicPhone" valid:"required"`
	ToClinicPhone      string    `json:"toClinicPhone" valid:"required"`
	FromEmail          string    `json:"fromEmail" valid:"required"`
	ToEmail            string    `json:"toEmail" valid:"required"`
	IsDirty            bool      `json:"isDirty" valid:"required"`
	CommunicationPhone string    `json:"-"`
	CommunicationText  string    `datastore:"CommunicationText,noindex"`
	IsSummary          bool      `json:"isSummary" valid:"required"`
	IsQR               bool      `json:"isQR" valid:"required"`
	SummaryText        string    `datastore:"SummaryText,noindex"`
	IsNew              bool      `json:"-"`
}

// AllReferrals ....

type AllReferrals struct {
	Referralls []DSReferral `json:"referrals"`
	CursorNext string       `json:"cursorNext"`
}

// SMS is returned after a text/sms message is posted to Twilio
type SMS struct {
	Sid         string  `json:"sid"`
	DateCreated string  `json:"date_created"`
	DateUpdate  string  `json:"date_updated"`
	DateSent    string  `json:"date_sent"`
	AccountSid  string  `json:"account_sid"`
	To          string  `json:"to"`
	From        string  `json:"from"`
	NumMedia    string  `json:"num_media"`
	Body        string  `json:"body"`
	Status      string  `json:"status"`
	Direction   string  `json:"direction"`
	APIVersion  string  `json:"api_version"`
	Price       *string `json:"price,omitempty"`
	URL         string  `json:"uri"`
}

// ParsedEmail ...
type ParsedEmail struct {
	Headers     map[string]string
	Body        map[string]string
	Attachments map[string][]byte
	RawRequest  *http.Request
}

// Parse ....
func (email *ParsedEmail) Parse() error {
	const _24K = 256 << 20
	err := email.RawRequest.ParseMultipartForm(_24K)
	if err != nil {
		return err
	}
	emails := email.RawRequest.MultipartForm.Value["email"]
	headers := email.RawRequest.MultipartForm.Value["headers"]
	if len(headers) > 0 {
		email.parseHeaders(headers[0])
	}
	if len(emails) > 0 {
		email.parseRawEmail(emails[0])
	}
	return nil
}

func (email *ParsedEmail) parseRawEmail(rawEmail string) {
	sections := strings.SplitN(rawEmail, "\n\n", 2)
	email.parseHeaders(sections[0])
	raw := parseMultipart(strings.NewReader(sections[1]), email.Headers["Content-Type"])
	for {
		emailPart, err := raw.NextPart()
		if err == io.EOF {
			return
		}
		rawEmailBody := parseMultipart(emailPart, emailPart.Header.Get("Content-Type"))
		if rawEmailBody != nil {
			for {
				emailBodyPart, err := rawEmailBody.NextPart()
				if err == io.EOF {
					break
				}
				header := emailBodyPart.Header.Get("Content-Type")
				email.Body[header] = string(readBody(emailBodyPart))
			}

		} else if emailPart.FileName() != "" {
			email.Attachments[emailPart.FileName()] = readBody(emailPart)
		} else {
			header := emailPart.Header.Get("Content-Type")
			email.Body[header] = string(readBody(emailPart))
		}
	}
}

func parseMultipart(body io.Reader, contentType string) *multipart.Reader {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Fatal(err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		return multipart.NewReader(body, params["boundary"])
	}
	return nil
}

func readBody(body io.Reader) []byte {
	raw, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatal(err)
	}
	return raw
}

func (email *ParsedEmail) parseHeaders(headers string) {
	splitHeaders := strings.Split(strings.TrimSpace(headers), "\n")
	for _, header := range splitHeaders {
		splitHeader := strings.SplitN(header, ": ", 2)
		email.Headers[splitHeader[0]] = splitHeader[1]
	}
}
