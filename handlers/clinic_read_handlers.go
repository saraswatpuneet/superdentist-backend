package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"go.opencensus.io/trace"
	"googlemaps.github.io/maps"
)

// GetPhysicalClinics ... after registering clinic main account we add multiple locations etc.
func GetPhysicalClinics(c *gin.Context) {
	log.Infof("Get all clinics associated with admin")
	ctx := c.Request.Context()
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Get all clinics associated with admin")
	defer span.End()
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	registeredClinics, err := clinicMetaDB.GetAllClinics(ctx, userEmail, userID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	responseData := contracts.GetClinicAddressResponse{
		ClinicDetails: registeredClinics,
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetClinicDoctors ... get doctors from specific clinic.
func GetClinicDoctors(c *gin.Context) {
	log.Infof("Get all doctors registered with specific physical clinic")
	addressID := c.Param("addressId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: "clinic address id not provided",
			},
		)
		return
	}
	ctx := c.Request.Context()
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Get all doctors registered for a clinic")
	defer span.End()
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	registeredDoctors, err := clinicMetaDB.GetClinicDoctors(ctx, userEmail, userID, addressID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   registeredDoctors,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetAllDoctors ... get all doctors working for admin.
func GetAllDoctors(c *gin.Context) {
	log.Infof("Get all doctors associated with admin businesses")
	ctx := c.Request.Context()
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	ctx, span := trace.StartSpan(ctx, "Get all doctors registered for a clinic")
	defer span.End()
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	registeredDoctors, err := clinicMetaDB.GetClinicDoctors(ctx, userEmail, userID, "")
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   registeredDoctors,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetNearbySpeialists ..... get near by clinics based on distance to current clinic
func GetNearbySpeialists(c *gin.Context) {
	log.Infof("Get specialists clinic in nearby give clinic")
	ctx := c.Request.Context()
	var nearbyRequest contracts.GetNearbySpecialists
	ctx, span := trace.StartSpan(ctx, "Get all clinics in close proximity to current clinic")
	defer span.End()
	if err := c.ShouldBindWith(&nearbyRequest, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}

	if nearbyRequest.ClinicAddessID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: "clinic address id not provided",
			},
		)
		return
	}
	dist := 20.0
	if nearbyRequest.SearchRadius == "" {
		nearbyRequest.SearchRadius = "20.0"
	}
	cursor := nearbyRequest.Cursor
	dist, _ = strconv.ParseFloat(nearbyRequest.SearchRadius, 64)
	userEmail, userID, gproject, err := getUserDetails(ctx, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	defer span.End()
	clinicMetaDB := datastoredb.NewClinicMetaHandler()
	err = clinicMetaDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	mapClient := gmaps.NewMapsHandler()
	err = mapClient.InitializeGoogleMapsAPIClient(ctx, gproject)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
	}
	collectClinics := make([]contracts.PhysicalClinicMapDetails, 0)
	currentClinic, _ := clinicMetaDB.GetSingleClinic(ctx, nearbyRequest.ClinicAddessID)
	loc := currentClinic.Location
	currentVerifiedPlaces := make(map[string]bool)

	if cursor == "" {
		nearbyClinics, err := clinicMetaDB.GetNearbySpecialist(ctx, userEmail, userID, nearbyRequest.ClinicAddessID, dist)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					constants.RESPONSE_JSON_DATA:   nil,
					constants.RESPONSDE_JSON_ERROR: err.Error(),
				},
			)
			return
		}
		for _, clinicAdd := range nearbyClinics {
			if clinicAdd.AddressID == nearbyRequest.ClinicAddessID || clinicAdd.Type == "General Dentist" {
				continue
			}
			var currentReturn contracts.PhysicalClinicMapDetails
			getClinicSearchLoc, err := mapClient.FindPlaceFromText(clinicAdd.Address)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						constants.RESPONSE_JSON_DATA:   nil,
						constants.RESPONSDE_JSON_ERROR: err.Error(),
					},
				)
			}
			if len(getClinicSearchLoc.Candidates) > 0 {
				currentReturn.GeneralDetails = getClinicSearchLoc.Candidates[0]
				currentVerifiedPlaces[currentReturn.GeneralDetails.PlaceID] = true
				clinicAdd.IsVerified = true
				currentReturn.VerifiedDetails = clinicAdd
				collectClinics = append(collectClinics, currentReturn)
			}
		}
	}
	currentMapLocation := maps.LatLng{
		Lat: loc.Lat,
		Lng: loc.Long,
	}
	currentSpeciality := nearbyRequest.Specialities
	if currentSpeciality == "" {
		currentSpeciality = "Specialist"
	}
	currentRadius := uint(dist * 1609.34) // in meters
	currentNonRegisteredNearby, pToken, err := mapClient.FindNearbyPlacesFromLocation(currentMapLocation, currentRadius, currentSpeciality, cursor, currentVerifiedPlaces)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
	}
	for _, clinicAdd := range currentNonRegisteredNearby {
		var currentReturn contracts.PhysicalClinicMapDetails

		currentReturn.GeneralDetails = clinicAdd
		currentReturn.VerifiedDetails = contracts.PhysicalClinicMapLocation{}
		currentReturn.VerifiedDetails.IsVerified = false
		collectClinics = append(collectClinics, currentReturn)

	}
	var responseData contracts.GetNearbyClinics
	responseData.ClinicAddresses = collectClinics
	responseData.Cursor = pToken
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}
