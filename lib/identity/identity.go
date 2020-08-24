package identity

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/internal"
	"github.com/superdentist/superdentist-backend/contracts"
	"google.golang.org/api/option"
)

// IDP ... client managing email/password related authorization
type IDP struct {
	projectID string
	client    *auth.Client
	fireApp   *firebase.App
}

// NewIDPEP .... intializes firebase auth which will do al sorts of authn/authz
func NewIDPEP(ctx context.Context, projectID string) (*IDP, error) {
	serviceAccountSD := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountSD == "" {
		return nil, fmt.Errorf("Missing service account file for backend server")
	}
	opt := option.WithCredentialsFile(serviceAccountSD)
	o := []option.ClientOption{opt}
	currentClient, err := auth.NewClient(ctx, &internal.AuthConfig{ProjectID: projectID, Opts: o})
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize identity client")
	}
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize identity client")
	}
	return &IDP{projectID: projectID, client: currentClient, fireApp: app}, nil
}

// SignUpUser ....
func (id *IDP) SignUpUser(ctx context.Context, user *contracts.UserRegistrationRequest) (*auth.UserRecord, error) {
	params := (&auth.UserToCreate{}).
		Email(user.Email).
		EmailVerified(false).
		PhoneNumber(user.PhoneNumber).
		Password(user.Password).
		DisplayName(user.FirstName + " " + user.LastName).
		Disabled(false)
	createResponse, err := id.client.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	return createResponse, nil
}
