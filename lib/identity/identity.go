// Package identity that handles signup, reset password, verification of email etc.
// This is an admin package be careful while using these functions .....
package identity

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/superdentist/superdentist-backend/lib/helpers"
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
	targetScopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	}
	currentCreds, _, err := helpers.ReadCredentialsFile(ctx, serviceAccountSD, targetScopes)
	opt := option.WithCredentials(currentCreds)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize identity client")
	}
	currentClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize identity client")
	}
	return &IDP{projectID: projectID, client: currentClient, fireApp: app}, nil
}

// ResetUserPassword ...
func (id *IDP) ResetUserPassword(ctx context.Context, email string) error {
	currentResetLink, err := id.client.PasswordResetLink(ctx, email)
	if err != nil {
		return err
	}
	// TODO: Implement SMTP server from GSuite/Others to send out custom emails
	// Would also need HTML template for the same
	fmt.Println(currentResetLink)
	return nil
}

// GetUserByEmail ...
func (id *IDP) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	currentUser, err := id.client.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return currentUser, nil
}

// GetUserByPhone ...
func (id *IDP) GetUserByPhone(ctx context.Context, phone string) (*auth.UserRecord, error) {
	currentUser, err := id.client.GetUserByPhoneNumber(ctx, phone)
	if err != nil {
		return nil, err
	}
	return currentUser, nil
}


// GetVerificationURL ...
func (id *IDP) GetVerificationURL(ctx context.Context, email string) (string, error) {
	verifyLink, err := id.client.EmailVerificationLink(ctx, email)
	if err != nil {
		return "", err
	}
	return verifyLink, nil
}
