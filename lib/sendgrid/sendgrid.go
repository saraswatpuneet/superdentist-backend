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

// SendEmailNotification ......
func (sgc *ClientSendGrid) SendEmailNotification(toEmail string) error {
	from := mail.NewEmail("SuperDentist Admin", constants.SD_ADMIN_EMAIL)
	subject := "Referral is created"
	to := mail.NewEmail("XYZPERSON", toEmail)
	plainText := "This is to test if referrals are created"
	message := mail.NewSingleEmail(from, subject, to, plainText, "")
	_, err := sgc.client.Send(message)
	if err != nil {
		return err
	}
	return nil
}
