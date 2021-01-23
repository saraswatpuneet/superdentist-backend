package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/storage"
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
	currentFavorites := currentClinic.Favorites
	if cursor == "" {
		nearbyClinics := make([]contracts.PhysicalClinicMapLocation, 0)
		nearbyClinics, err = clinicMetaDB.GetNearbySpecialist(ctx, userEmail, userID, nearbyRequest.ClinicAddessID, dist)
		if err != nil {
			log.Infof("no nearby clinics found: %v", err.Error())
		}
		for _, clinicAdd := range nearbyClinics {
			if clinicAdd.AddressID == nearbyRequest.ClinicAddessID || clinicAdd.Type == "dentist" {
				continue
			}
			if Find(currentFavorites, clinicAdd.PlaceID) {
				continue
			}
			var currentReturn contracts.PhysicalClinicMapDetails
			getClinicSearchLoc, err := mapClient.FindPlaceFromID(clinicAdd.PlaceID)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						constants.RESPONSE_JSON_DATA:   nil,
						constants.RESPONSDE_JSON_ERROR: err.Error(),
					},
				)
			}

			currentReturn.GeneralDetails = *getClinicSearchLoc
			if clinicAdd.Specialty != nil {
				for _, sp := range clinicAdd.Specialty {
					currentReturn.GeneralDetails.Types = append(currentReturn.GeneralDetails.Types, sp)
				}
			}
			currentVerifiedPlaces[currentReturn.GeneralDetails.PlaceID] = true
			clinicAdd.IsVerified = true
			currentReturn.VerifiedDetails = clinicAdd
			collectClinics = append(collectClinics, currentReturn)
		}
	}
	currentMapLocation := maps.LatLng{
		Lat: loc.Lat,
		Lng: loc.Long,
	}
	currentSpeciality := nearbyRequest.Specialties
	if currentSpeciality == "" {
		currentSpeciality = "specialist"
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
		if Find(currentFavorites, clinicAdd.PlaceID) {
			continue
		}
		var currentReturn contracts.PhysicalClinicMapDetails

		currentReturn.GeneralDetails = clinicAdd
		currentReturn.VerifiedDetails = contracts.PhysicalClinicMapLocation{}
		currentReturn.VerifiedDetails.IsVerified = false
		collectClinics = append(collectClinics, currentReturn)

	}
	var responseData contracts.GetNearbyClinics
	responseData.ClinicAddresses = collectClinics
	for i, clinic := range collectClinics {
		for key, value := range gmaps.SPECIALITYMAP {
			if strings.Contains(strings.ToLower(clinic.GeneralDetails.Name), key) {
				clinic.GeneralDetails.Types[0] = value
				break
			}
		}
		collectClinics[i] = clinic
	}
	responseData.Cursor = pToken
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// AddFavoriteClinics ...
func AddFavoriteClinics(c *gin.Context) {
	log.Infof("Add Favorite Clinics")
	ctx := c.Request.Context()
	var favoriteAdd contracts.AddFavoriteClinics
	ctx, span := trace.StartSpan(ctx, "Get all clinics in close proximity to current clinic")
	defer span.End()
	if err := c.ShouldBindWith(&favoriteAdd, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	addressID := c.Param("addressId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing clinic address id"),
			},
		)
		return
	}
	_, userID, gproject, err := getUserDetails(ctx, c.Request)
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
	currentClinic, err := clinicMetaDB.GetSingleClinic(ctx, addressID)
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
	currentClinic.Favorites = append(currentClinic.Favorites, favoriteAdd.PlaceIDs...)
	err = clinicMetaDB.UpdatePhysicalAddessressToClinic(ctx, userID, *currentClinic)
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
	err = clinicMetaDB.UpdateNetworkForFavoritedClinic(ctx, *currentClinic)
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
	storageC := storage.NewStorageHandler()
	err = storageC.InitializeStorageClient(ctx, gproject)
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
	for _, fav := range currentClinic.Favorites {
		if currentClinic.Type == "dentist" {
			GenerateQRAndStore(ctx, currentClinic.PlaceID, currentClinic.PlaceID, fav, storageC)

		} else {
			GenerateQRAndStore(ctx, currentClinic.PlaceID, fav, currentClinic.PlaceID, storageC)
		}

	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "Added favorite places to current clinic",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetAllQRZip ...
func GetAllQRZip(c *gin.Context) {
	log.Infof("Add Favorite Clinics")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Get all clinics in close proximity to current clinic")
	defer span.End()
	addressID := c.Param("placeId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing clinic address id"),
			},
		)
		return
	}
	userEmail, _, gproject, err := getUserDetails(ctx, c.Request)
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
	if !strings.Contains(userEmail, "@superdentist.io") {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Not allowed to access this api"),
			},
		)
		return
	}
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
	currentClinic, err := clinicMetaDB.GetSingleClinicViaPlace(ctx, addressID)
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
	storageC := storage.NewStorageHandler()
	err = storageC.InitializeStorageClient(ctx, gproject)
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
	err = storageC.ZipFile(ctx, currentClinic.PlaceID, constants.SD_QR_BUCKET)
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

	zipReader, err := storageC.DownloadAsZip(ctx, currentClinic.PlaceID, constants.SD_QR_BUCKET)
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
	fileNameDefault := currentClinic.Name + ".zip"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileNameDefault))
	c.Header("Content-Type", "application/zip")
	clinicMetaDB.Close()
	if _, err := io.Copy(c.Writer, zipReader); err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
}

// GetFavoriteClinics ...
func GetFavoriteClinics(c *gin.Context) {
	log.Infof("Get specialists clinic in nearby give clinic")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Get all clinics favorited by current clinic")
	defer span.End()
	addressID := c.Param("addressId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing clinic address id"),
			},
		)
		return
	}
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
	currentClinic, _ := clinicMetaDB.GetSingleClinic(ctx, addressID)
	currentFavorites := currentClinic.Favorites
	favoriteClinics, err := clinicMetaDB.GetFavoriteSpecialists(ctx, userEmail, userID, currentFavorites)
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
	storageC := storage.NewStorageHandler()
	err = storageC.InitializeStorageClient(ctx, gproject)
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
	isDentist := currentClinic.Type == "dentist"
	for _, clinicAdd := range favoriteClinics {
		var currentReturn contracts.PhysicalClinicMapDetails
		qrpng := make([]byte, 0)
		if isDentist {
			qrpng = GetQRPNGBytes(ctx, currentClinic.PlaceID, clinicAdd.PlaceID)
		} else {
			qrpng = GetQRPNGBytes(ctx, clinicAdd.PlaceID, currentClinic.PlaceID)

		}
		pngQRBase := base64.StdEncoding.EncodeToString(qrpng)

		getClinicSearchLoc, err := mapClient.FindPlaceFromID(clinicAdd.PlaceID)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					constants.RESPONSE_JSON_DATA:   nil,
					constants.RESPONSDE_JSON_ERROR: err.Error(),
				},
			)
		}
		currentReturn.GeneralDetails = *getClinicSearchLoc
		currentReturn.VerifiedDetails = clinicAdd
		currentReturn.QRCode = pngQRBase
		collectClinics = append(collectClinics, currentReturn)

	}

	var responseData contracts.GetFavClinics
	responseData.ClinicAddresses = collectClinics
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// GetNetworkClinics ...
func GetNetworkClinics(c *gin.Context) {
	log.Infof("Get specialists clinic in nearby give clinic")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Get all clinics favorited by current clinic")
	defer span.End()
	addressID := c.Param("addressId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing clinic address id"),
			},
		)
		return
	}
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
	currentClinic, _ := clinicMetaDB.GetSingleClinic(ctx, addressID)
	currentFavorites, err := clinicMetaDB.GetNetworkClincs(ctx, currentClinic.PlaceID)
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
	favoriteClinics, err := clinicMetaDB.GetFavoriteSpecialists(ctx, userEmail, userID, currentFavorites)
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
	for _, clinicAdd := range favoriteClinics {
		var currentReturn contracts.PhysicalClinicMapDetails
		getClinicSearchLoc, err := mapClient.FindPlaceFromID(clinicAdd.PlaceID)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					constants.RESPONSE_JSON_DATA:   nil,
					constants.RESPONSDE_JSON_ERROR: err.Error(),
				},
			)
		}
		currentReturn.GeneralDetails = *getClinicSearchLoc
		currentReturn.VerifiedDetails = clinicAdd
		collectClinics = append(collectClinics, currentReturn)

	}

	var responseData contracts.GetFavClinics
	responseData.ClinicAddresses = collectClinics
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   responseData,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// RemoveFavoriteClinics ...
func RemoveFavoriteClinics(c *gin.Context) {
	log.Infof("Remove Favorite Clinics")
	ctx := c.Request.Context()
	var favoriteAdd contracts.AddFavoriteClinics
	ctx, span := trace.StartSpan(ctx, "Get all clinics in close proximity to current clinic")
	defer span.End()
	if err := c.ShouldBindWith(&favoriteAdd, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	addressID := c.Param("addressId")
	if addressID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing clinic address id"),
			},
		)
		return
	}
	_, userID, gproject, err := getUserDetails(ctx, c.Request)
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
	currentClinic, err := clinicMetaDB.GetSingleClinic(ctx, addressID)
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
	updatedFavorites := make([]string, 0)
	for _, favID := range currentClinic.Favorites {
		exists := Find(favoriteAdd.PlaceIDs, favID)
		if !exists {
			updatedFavorites = append(updatedFavorites, favID)
		} else {
			clinicMetaDB.RemoveNetworkForFavoritedClinic(ctx, favID, currentClinic.PlaceID)

		}
	}
	currentClinic.Favorites = updatedFavorites
	err = clinicMetaDB.UpdatePhysicalAddessressToClinic(ctx, userID, *currentClinic)
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
		constants.RESPONSE_JSON_DATA:   "Remove places from favorites",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	clinicMetaDB.Close()
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// GenerateQRAndStore ....
func GenerateQRAndStore(ctx context.Context, folder string, from string, to string, storageC *storage.Client) []byte {
	url := "from+" + from + "+to+" + to
	urlEncoded, err := encryptAndEncode(url)
	if err != nil {
		log.Errorf("failed to encode qr url: %v", err.Error())
	}
	currentURL := fmt.Sprintf(constants.QR_URL_CODE, urlEncoded)
	png, err := qrcode.Encode(currentURL, qrcode.Medium, 256)
	if err != nil {
		log.Errorf("failed to create qr image: %v", err.Error())
		return nil
	}
	fileName := from + to
	bucketPath := folder + "/" + fileName + ".png"
	buckerW, err := storageC.UploadQRtoGCS(ctx, bucketPath)
	if err != nil {
		log.Errorf("failed to create bucket image: %v", err.Error())
		return nil
	}
	_, err = io.Copy(buckerW, bytes.NewReader(png))
	if err != nil {
		log.Errorf("failed to upload qr image to bucket: %v", err.Error())
		return nil
	}
	buckerW.Close()
	return png
}

// GetQRPNGBytes ....
func GetQRPNGBytes(ctx context.Context, from string, to string) []byte {
	url := "from+" + from + "+to+" + to
	urlEncoded, err := encryptAndEncode(url)
	if err != nil {
		log.Errorf("failed to encode qr url: %v", err.Error())
	}
	currentURL := fmt.Sprintf(constants.QR_URL_CODE, urlEncoded)
	png, err := qrcode.Encode(currentURL, qrcode.Medium, 256)
	if err != nil {
		log.Errorf("failed to create qr image: %v", err.Error())
		return nil
	}
	return png
}

func encryptAndEncode(toencode string) (string, error) {
	nonce := make([]byte, global.Options.GCMQR.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(global.Options.GCMQR.Seal(nonce, nonce, []byte(toencode), nil)), nil
}
