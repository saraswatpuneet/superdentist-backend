package gmaps

import (
	"context"
	"fmt"
	"os"

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
	placesFromTextReq := maps.TextSearchRequest{
		Query: placeText,
		Type:  maps.PlaceTypeDentist,
	}
	placesSearchResponse, err := gm.client.TextSearch(ctx, &placesFromTextReq)
	if err != nil {
		return maps.PlacesSearchResponse{}, err
	}
	return placesSearchResponse, nil
}

// FindNearbyPlacesFromLocation ....
func (gm *ClientGMaps) FindNearbyPlacesFromLocation(location maps.LatLng, radius uint, keyword string, toke string, ignorePlaces map[string]bool) (map[string]maps.PlacesSearchResult, string, error) {
	ctx := context.Background()
	nearbyClinicsMap := make(map[string]maps.PlacesSearchResult)

	placesFromTextReq := maps.NearbySearchRequest{
		Location: &location,
		Radius:   radius,
		Keyword:  keyword,
		Type:     maps.PlaceTypeDentist,
	}
	placesSearchResponse, err := gm.client.NearbySearch(ctx, &placesFromTextReq)
	if err != nil {
		return nil, "", err
	}
	for _, place := range placesSearchResponse.Results {
		if _, ok := ignorePlaces[place.PlaceID]; !ok {
			nearbyClinicsMap[place.PlaceID] = place
		}
	}
	return nearbyClinicsMap, placesSearchResponse.NextPageToken, nil
}
