package sms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/kevinburke/twilio-go"
)

// ClientSMS ....
type ClientSMS struct {
	client     *twilio.Client
	httpClient *http.Client
	twiSID     string
	twiAuth    string
}

// NewSMSClient return new database action
func NewSMSClient() *ClientSMS {
	return &ClientSMS{client: nil, httpClient: nil}
}

// InitializeSMSClient ....
func (twiC *ClientSMS) InitializeSMSClient() error {
	twiSID := os.Getenv("TWI_SID")
	twiAUTH := os.Getenv("TWI_AUTH")
	if twiSID == "" || twiAUTH == "" {
		return fmt.Errorf("Sendgrid api key not found")
	}
	client := twilio.NewClient(twiSID, twiAUTH, nil)
	twiC.client = client
	twiC.twiAuth = twiAUTH
	twiC.twiSID = twiSID
	defaultTimeout := 30*time.Second + 500*time.Millisecond

	twiC.httpClient = &http.Client{
		Timeout: defaultTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Go's http.DefaultClient allows 10 redirects before returning an
			// an error. We have mimicked this default behavior.s
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
	return nil
}

// GetMedia ....
func (twiC *ClientSMS) GetMedia(ctx context.Context, url string) (string, *io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.SetBasicAuth(twiC.twiSID, twiC.twiAuth)
	if err != nil {
		return "", nil, err
	}
	req = req.WithContext(ctx)
	resp, err := twiC.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	// https://www.twilio.com/docs/api/rest/accepted-mime-types#supported
	ctype := resp.Header.Get("Content-Type")
	imageName, _ := uuid.NewUUID()
	switch ctype {
	case "image/jpeg":
		return imageName.String() + ".jpeg", &resp.Body, nil
	case "image/gif":
		return imageName.String() + ".gif", &resp.Body, nil
	case "image/png":
		return imageName.String() + ".png", &resp.Body, nil
	default:
		return "", nil, fmt.Errorf("twilio: Unknown content-type %s", ctype)
	}
}

// SendSMS ....
func (twiC *ClientSMS) SendSMS(fromPhone string, toPhone string, messageBody string) error {
	_, err := twiC.client.Messages.SendMessage(fromPhone, toPhone, messageBody, nil)
	if err != nil {
		return err
	}
	return nil
}
