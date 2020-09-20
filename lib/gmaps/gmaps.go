package gmaps

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"googlemaps.github.io/maps"
)

// ClientGMaps ....
type ClientGMaps struct {
	projectID string
	client    *maps.Client
}

// NewMapsHandler return new database action
func NewMapsHandler() *ClientGMaps {
	return &ClientGMaps{projectID: "", client: nil}
}

// InitializeGoogleMapsAPIClient ....
func (gm *ClientGMaps) InitializeGoogleMapsAPIClient(ctx context.Context, projectID string) error {
	gcpAPIKey := os.Getenv("GCP_API_KEY")
	if gcpAPIKey == "" {
		return fmt.Errorf("Failed to get api key superdentist backend")
	}
	gmClient, err := maps.NewClient(maps.WithAPIKey(gcpAPIKey))
	if err != nil {
		return err
	}
	gm.client = gmClient
	gm.projectID = projectID
	return nil
}

// FindPlaceFromText ....
func (gm *ClientGMaps) FindPlaceFromText(placeText string) (maps.FindPlaceFromTextResponse, error) {
	ctx := context.Background()
	placesFromTextReq := maps.FindPlaceFromTextRequest{
		Input:     placeText,
		InputType: maps.FindPlaceFromTextInputTypeTextQuery,
		Fields:    []maps.PlaceSearchFieldMask{maps.PlaceSearchFieldMaskName, maps.PlaceSearchFieldMaskFormattedAddress},
	}
	placesSearchResponse, err := gm.client.FindPlaceFromText(ctx, &placesFromTextReq)
	if err != nil {
		return maps.FindPlaceFromTextResponse{}, err
	}
	return placesSearchResponse, nil
}

// FindPlacesFromText ....
func (gm *ClientGMaps) FindPlacesFromText(placeText string) (maps.PlacesSearchResponse, error) {
	ctx := context.Background()
	randomPageToken, _ := uuid.NewUUID()
	placesFromTextReq := maps.TextSearchRequest{
		Query:     placeText,
		PageToken: randomPageToken.String(),
	}
	placesSearchResponse, err := gm.client.TextSearch(ctx, &placesFromTextReq)
	if err != nil {
		return maps.PlacesSearchResponse{}, err
	}
	return placesSearchResponse, nil
}
