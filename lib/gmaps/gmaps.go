package gmaps

import (
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
