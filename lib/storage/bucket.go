package storage

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/option"
)

// Client ....
type Client struct {
	projectID string
	client    *storage.Client
}

// NewStorageHandler return new database action
func NewStorageHandler() *Client {
	return &Client{projectID: "", client: nil}
}

// InitializeStorageClient ...........
func (sc *Client) InitializeStorageClient(ctx context.Context, projectID string) error {
	serviceAccountSD := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountSD == "" {
		return fmt.Errorf("Failed to get right credentials for superdentist backend")
	}
	targetScopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	}
	currentCreds, _, err := helpers.ReadCredentialsFile(ctx, serviceAccountSD, targetScopes)
	if err != nil {
		return err
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(currentCreds))
	if err != nil {
		return err
	}
	sc.client = client
	sc.projectID = projectID
	return nil
}

// CreateBucket  ...........
func (sc *Client) CreateBucket(ctx context.Context, bucketName string) (*storage.BucketHandle, error) {
	bkt := sc.client.Bucket(bucketName)
	exists, err := bkt.Attrs(ctx)
	if err != nil && exists == nil {
		if err := bkt.Create(ctx, sc.projectID, nil); err != nil {
			return nil, err
		}
	}
	return bkt, nil
}

// UploadToGCS ....
func (sc *Client) UploadToGCS(ctx context.Context, fileName string) (*storage.Writer, error) {
	currentBucket, err := sc.CreateBucket(ctx, constants.SD_REFERRAL_BUCKET)
	if err != nil {
		return nil, err
	}
	bucketWriter := currentBucket.Object(fileName).NewWriter(ctx)
	return bucketWriter, nil
}
