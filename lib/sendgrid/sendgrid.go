package sendgrid

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/superdentist/superdentist-backend/constants"
)

// ClientSendGrid ....
type ClientSendGrid struct {
	client *sendgrid.Client
}

// NewSendGridClient return new database action
func NewSendGridClient() *ClientSendGrid {
	return &ClientSendGrid{client: nil}
}

// InitializeSendGridClient ....
func (sgc *ClientSendGrid) InitializeSendGridClient() error {
	sgAPIKey := os.Getenv("SENDGRID_API_KEY")
	if sgAPIKey == "" {
		return fmt.Errorf("Sendgrid api key not found")
	}
	client := sendgrid.NewSendClient(sgAPIKey)
	sgc.client = client
	return nil
}

// SendEmailNotificationPatient ......
func (sgc *ClientSendGrid) SendEmailNotificationPatient(pemail string,
	pname string,
	spname string,
	spphone string,
	refid string,
	spaddress string,
	comments []string) error {
	mailSetup := mail.NewV3Mail()
	from := mail.NewEmail("SuperDentist Admin", constants.SD_ADMIN_EMAIL)
	mailSetup.SetFrom(from)
	mailSetup.SetTemplateID(constants.SD_PATIENT_REF_CONF)
	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(pname, pemail),
	}
	p.AddTos(tos...)
	p.SetDynamicTemplateData("subject", "Your Referral to "+spname)
	p.SetDynamicTemplateData("pname", pname)
	p.SetDynamicTemplateData("refid", refid)
	p.SetDynamicTemplateData("spname", spname)
	p.SetDynamicTemplateData("address", spaddress)
	p.SetDynamicTemplateData("phone", spphone)
	p.SetDynamicTemplateData("comments", comments)
	mailSetup.AddPersonalizations(p)
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(mailSetup)
	request.Body = Body
	_, err := sendgrid.API(request)
	if err != nil {
		return err
	}
	return nil
}

// SendEmailNotificationSpecialist ......
func (sgc *ClientSendGrid) SendEmailNotificationSpecialist(spemail string,
	pname string,
	spname string,
	pphone string,
	refid string,
	rdate string,
	comments []string) error {
	mailSetup := mail.NewV3Mail()
	from := mail.NewEmail("SuperDentist Admin", constants.SD_ADMIN_EMAIL)
	mailSetup.SetFrom(from)
	mailSetup.SetTemplateID(constants.SD_SPECIALIZT_REF_CONF)
	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(spname, spemail),
	}
	p.AddTos(tos...)
	p.SetDynamicTemplateData("subject", "You have recieved a New Patient Referral on SuperDentist! Referral ID: "+refid)
	p.SetDynamicTemplateData("pname", pname)
	p.SetDynamicTemplateData("refid", refid)
	p.SetDynamicTemplateData("spname", spname)
	p.SetDynamicTemplateData("pphone", pphone)
	p.SetDynamicTemplateData("rdate", rdate)
	p.SetDynamicTemplateData("comments", comments)
	mailSetup.AddPersonalizations(p)
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(mailSetup)
	request.Body = Body
	_, err := sendgrid.API(request)
	if err != nil {
		return err
	}
	return nil
}

// SendCompletionEmailToGD ......
func (sgc *ClientSendGrid) SendCompletionEmailToGD(gdemail string, gdname string,
	pname string,
	spname string,
	pphone string,
	refid string,
	cdate string,
	comments []string) error {
	mailSetup := mail.NewV3Mail()
	from := mail.NewEmail("SuperDentist Admin", constants.SD_ADMIN_EMAIL)
	mailSetup.SetFrom(from)
	mailSetup.SetTemplateID(constants.GD_REFERRAL_COMPLETED)
	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(spname, gdemail),
	}
	p.AddTos(tos...)
	p.SetDynamicTemplateData("subject", "Your Patient Referral has been Completed on SuperDentist! Referral ID: "+refid)
	p.SetDynamicTemplateData("pname", pname)
	p.SetDynamicTemplateData("refid", refid)
	p.SetDynamicTemplateData("spname", spname)
	p.SetDynamicTemplateData("pphone", pphone)
	p.SetDynamicTemplateData("cdate", cdate)
	p.SetDynamicTemplateData("comments", comments)
	mailSetup.AddPersonalizations(p)
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(mailSetup)
	request.Body = Body
	_, err := sendgrid.API(request)
	if err != nil {
		return err
	}
	return nil
}

// SendClinicNotification ....
func (sgc *ClientSendGrid) SendClinicNotification(cemail string, cname string, pname string, refid string) error {
	mailSetup := mail.NewV3Mail()
	from := mail.NewEmail("SuperDentist Admin", constants.SD_ADMIN_EMAIL)
	mailSetup.SetFrom(from)
	mailSetup.SetTemplateID(constants.CLINIC_NOTIFICATION_NEW)
	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(cname, cemail),
	}
	p.AddTos(tos...)
	p.SetDynamicTemplateData("subject", "You have a new notification on SuperDentist! Referral ID: "+refid)
	p.SetDynamicTemplateData("pname", pname)
	p.SetDynamicTemplateData("refid", refid)
	p.SetDynamicTemplateData("cname", cname)
	mailSetup.AddPersonalizations(p)
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(mailSetup)
	request.Body = Body
	_, err := sendgrid.API(request)
	if err != nil {
		return err
	}
	return nil
}
