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

// PatientDentalInsurance ....
type PatientDentalInsurance struct {
	Company  string `json:"company"`
	MemberID string `json:"memberID"`
}

// PatientMedicalInsurance ....
type PatientMedicalInsurance struct {
	Company     string `json:"company"`
	GroupNumber string `json:"groupNumber"`
	MemberID    string `json:"memberID"`
}

// Patient ....
type Patient struct {
	FirstName        string                    `json:"firstName" valid:"required"`
	LastName         string                    `json:"lastName" valid:"required"`
	Dob              DOB                       `json:"dob"`
	Email            string                    `json:"email" valid:"required"`
	Phone            string                    `json:"phone" valid:"required"`
	SSN              string                    `json:"ssn"`
	DentalInsurance  []PatientDentalInsurance  `datastore:"dentalInsurance,noindex"`
	MedicalInsurance []PatientMedicalInsurance `datastore:"medicalInsurance,noindex"`
}

// ChatBox ....
type ChatBox string

// GDCBox ....
const GDCBox ChatBox = "c2c"

// SPCBox ....
const SPCBox ChatBox = "c2p"

// Comment .....
type Comment struct {
	MessageID string   `json:"messageId"`
	TimeStamp int64    `json:"timeStamp"`
	Text      string   `json:"text" valid:"required"`
	Channel   ChatBox  `json:"channel" valid:"required"`
	UserID    string   `json:"userId" valid:"required"`
	Files     []string `json:"file"`
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
	FromPlaceID   string    `json:"fromPlaceId"`
	ToPlaceID     string    `json:"toPlaceId"`
	Status        Status    `json:"status" valid:"required"`
	Comments      []Comment `json:"comments"`
	Reasons       []string  `json:"reasons"`
	History       []string  `json:"history"`
	Tooth         []string  `json:"tooth"`
	IsSummary     bool      `json:"isSummary"`
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
