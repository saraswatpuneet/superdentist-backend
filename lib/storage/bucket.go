package storage

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/lib/helpers"
	"google.golang.org/api/iterator"
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

// UploadQRtoGCS ....
func (sc *Client) UploadQRtoGCS(ctx context.Context, fileName string) (*storage.Writer, error) {
	currentBucket, err := sc.CreateBucket(ctx, constants.SD_QR_BUCKET)
	if err != nil {
		return nil, err
	}
	bucketWriter := currentBucket.Object(fileName).NewWriter(ctx)
	return bucketWriter, nil
}

// ZipFile ....ZipFile
func (sc *Client) ZipFile(ctx context.Context, folderPath string, bucket string) error {
	currentBucket, err := sc.CreateBucket(ctx, bucket)
	if err != nil {
		return err
	}
	currentFolder := fmt.Sprintf("%v/", folderPath)
	storageQuery := &storage.Query{Prefix: currentFolder, Delimiter: "/"}

	zipFileURI := currentFolder + "zip/zipped.zip"
	err = currentBucket.Object(zipFileURI).Delete(ctx)
	storageWriter := currentBucket.Object(zipFileURI).NewWriter(ctx)
	storageWriter.ContentType = "application/zip"
	defer storageWriter.Close()
	zipWriter := zip.NewWriter(storageWriter)
	// go through each file in the prefix
	refDocsIterator := currentBucket.Objects(ctx, storageQuery)
	objects := []*storage.ObjectAttrs{}

	for {
		objectAttrs, err := refDocsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		objects = append(objects, objectAttrs)
	}
	if len(objects) < 1 {
		return fmt.Errorf("No document is associated with current referral")
	}
	for _, obj := range objects {
		log.Printf("Packing file %v of size %v to zip file", obj.Name, obj.Size)

		storageReader, rerr := currentBucket.Object(obj.Name).NewReader(ctx)
		if rerr != nil {
			return err
		}

		// make all paths relative
		relativeFilename, rerr := filepath.Rel("/"+currentFolder, "/"+obj.Name)
		if rerr != nil {
			return err
		}

		// add filename to zip
		zipFile, zerr := zipWriter.Create(relativeFilename)
		if zerr != nil {
			return err
		}

		// copy from storage reader to zip writer
		_, cerr := io.Copy(zipFile, storageReader)
		if cerr != nil {
			return err
		}

		storageReader.Close()
	}

	err = zipWriter.Close()
	if err != nil {
		return err
	}
	storageWriter.Close() // we should be uploaded by here
	return nil
}

// DownloadAsZip ....
func (sc *Client) DownloadAsZip(ctx context.Context, folderPath string, bucket string) (*storage.Reader, error) {
	currentBucket, err := sc.CreateBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}
	zipFileURI := folderPath + "/zip/zipped.zip"
	storageReader, err := currentBucket.Object(zipFileURI).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return storageReader, nil
}

// DownloadSingleFile ....
func (sc *Client) DownloadSingleFile(ctx context.Context, folderPath string, bucket string, fileName string) (*storage.Reader, error) {
	currentBucket, err := sc.CreateBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}
	zipFileURI := folderPath + "/" + fileName
	storageReader, err := currentBucket.Object(zipFileURI).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return storageReader, nil
}
