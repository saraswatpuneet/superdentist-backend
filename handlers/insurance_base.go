package handlers

import (

	//"github.com/otiai10/gosseract/v2"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"go.opencensus.io/trace"
)

// GetAllPracticeCodesCats ....
func GetAllPracticeCodesCats(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Get Insurance Stuff")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Return all insurance codes")
	defer span.End()
	jsonFile, err := os.Open("./codes/d_codes.json")
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
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	codeMapping := make(map[string]interface{}, 0)
	err = json.Unmarshal(jsonBytes, &codeMapping)
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
		constants.RESPONSE_JSON_DATA:   codeMapping,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetAllDentalInsurances ....
func GetAllDentalInsurances(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Get Insurance Stuff")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Return all insurance codes")
	defer span.End()
	jsonFile, err := os.Open("./insurance/dental_insurances.json")
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
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	codeMapping := make([]map[string]interface{}, 0)
	err = json.Unmarshal(jsonBytes, &codeMapping)
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
		constants.RESPONSE_JSON_DATA:   codeMapping,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetAllMedicalInsurances ....
func GetAllMedicalInsurances(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Get Insurance Stuff")
	ctx := c.Request.Context()
	ctx, span := trace.StartSpan(ctx, "Return all insurance codes")
	defer span.End()
	jsonFile, err := os.Open("./insurance/dental_insurances.json")
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
	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	codeMapping := make([]map[string]interface{}, 0)
	err = json.Unmarshal(jsonBytes, &codeMapping)
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
		constants.RESPONSE_JSON_DATA:   codeMapping,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}
