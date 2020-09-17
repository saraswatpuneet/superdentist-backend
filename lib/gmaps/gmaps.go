package gmaps

import (
	"context"
	"fmt"
	"os"

	"googlemaps.github.io/maps"
)

type gmClincs struct {
	projectID string
	client    *maps.Client
}

//NewMapsHander return new database action
func NewMapsHander() *gmClincs {
	return &gmClincs{projectID: "", client: nil}
}

// InitializeGoogleMapsAPIClient ....
func (db *gmClincs) InitializeGoogleMapsAPIClient(ctx context.Context, projectID string) error {
	gcpAPIKey := os.Getenv("GCP_API_KEY")
	if gcpAPIKey == "" {
		return fmt.Errorf("Failed to get api key superdentist backend")
	}
	gmClient, err := maps.NewClient(maps.WithAPIKey(gcpAPIKey))
	if err != nil {
		return err
	}
	db.client = gmClient
	db.projectID = projectID
	return nil
}
