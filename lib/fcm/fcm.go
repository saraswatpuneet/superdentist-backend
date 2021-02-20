package fcm

import (
	"context"
	"fmt"
	"os"

	"github.com/NaySoftware/go-fcm"
)

// ClientFCM ....
type ClientFCM struct {
	projectID string
	client    *fcm.FcmClient
}

// NewFCMHanlder return new database action
func NewFCMHanlder() *ClientFCM {
	return &ClientFCM{projectID: "", client: nil}
}

// InitializeFCMClient ....
func (cfc *ClientFCM) InitializeFCMClient(ctx context.Context, projectID string) error {
	gcpAPIKey := os.Getenv("GCP_API_KEY")
	if gcpAPIKey == "" {
		return fmt.Errorf("Failed to get api key superdentist backend")
	}
	fcmClient := fcm.NewFcmClient(gcpAPIKey)
	cfc.client = fcmClient
	cfc.projectID = projectID
	return nil
}

// SendNotificationToUser ...
func (cfc *ClientFCM) SendNotificationToUser(ctx context.Context, topic string, message map[string]string) error {
	topicFull := fmt.Sprintf("/topics/%s", topic)
	cfc.client.NewFcmMsgTo(topicFull, message)
	status, err := cfc.client.Send()
	if err != nil {
		return err
	}
	if !status.Ok {
		return fmt.Errorf("Error in sending notification: %s", status.Err)
	}
	return nil
}
