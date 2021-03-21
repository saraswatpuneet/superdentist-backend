package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"gopkg.in/ugjka/go-tz.v2/tz"

	"code.sajari.com/docconv"
	pe "github.com/DusanKasan/parsemail"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/global"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/gmaps"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/sendgrid"
	"github.com/superdentist/superdentist-backend/lib/sms"
	"github.com/superdentist/superdentist-backend/lib/storage"
)

// AddCommentsToReferral ...
func AddCommentsToReferral(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Add comments to Referral")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")

	var referralDetails contracts.ReferralComments
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	if err := c.ShouldBindWith(&referralDetails, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}
	updatedComm, err := ProcessComments(ctx, gproject, referralID, referralDetails)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   updatedComm,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetAllMessages ....
func GetAllMessages(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Add comments to Referral")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")
	channel := c.Query("channel")
	if referralID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing referral ID"),
			},
		)
		return
	}
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	allComments, err := dsRefC.GetMessagesAllWithChannel(ctx, referralID, channel)
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
		constants.RESPONSE_JSON_DATA:   allComments,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetOneMessage ....
func GetOneMessage(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Update Referral Status")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")
	messageID := c.Param("messageId")
	if referralID == "" || messageID == "" {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Missing referral/message ID"),
			},
		)
		return
	}
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	oneComment, err := dsRefC.GetOneMessage(ctx, referralID, messageID)
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
		constants.RESPONSE_JSON_DATA:   oneComment,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// UpdateReferralStatus ...
func UpdateReferralStatus(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Update Referral Status")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")

	var referralDetails contracts.ReferralStatus
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	if err := c.ShouldBindWith(&referralDetails, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad data sent to backened"),
			},
		)
		return
	}

	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferral, err := dsRefC.GetReferral(ctx, referralID)
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
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
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
	toClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, dsReferral.ToPlaceID)
	if dsReferral.ToAddressID == "" {
		if err == nil && toClinic != nil && toClinic.AddressID != "" {
			dsReferral.ToPlaceID = toClinic.PlaceID
			dsReferral.ToClinicName = toClinic.Name
			dsReferral.ToClinicAddress = toClinic.Address
			dsReferral.ToEmail = toClinic.EmailAddress
			dsReferral.ToClinicPhone = toClinic.PhoneNumber
			dsReferral.ToAddressID = toClinic.AddressID
		}
	}
	dsReferral.Status = referralDetails.Status

	if err == nil && toClinic != nil {
		zone, _ := tz.GetZone(tz.Point{
			Lon: toClinic.Location.Long, Lat: toClinic.Location.Lat,
		})

		location, _ := time.LoadLocation(zone[0])
		dsReferral.ModifiedOn = time.Now().In(location)

	} else {
		dsReferral.ModifiedOn = time.Now()
	}
	err = dsRefC.CreateReferral(ctx, *dsReferral)
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
	if strings.ToLower(dsReferral.Status.SPStatus) == "completed" || strings.ToLower(dsReferral.Status.SPStatus) == "complete" {
		sgClient := sendgrid.NewSendGridClient()
		err = sgClient.InitializeSendGridClient()
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
		y, m, d := dsReferral.ModifiedOn.Date()
		dateString := fmt.Sprintf("%d-%d-%d", y, int(m), d)
		sendPatientComments := make([]string, 0)
		comments, err := dsRefC.GetMessagesAll(ctx, dsReferral.ReferralID)
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
		for _, comment := range comments {
			if comment.Channel == contracts.GDCBox && dsReferral.ToEmail != "" && comment.UserID == dsReferral.ToEmail {
				sendPatientComments = append(sendPatientComments, comment.Text)
			}
		}
		err = sgClient.SendCompletionEmailToGD(dsReferral.FromEmail, dsReferral.FromClinicName,
			dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName, dsReferral.PatientPhone, dsReferral.ReferralID, dateString, sendPatientComments)
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// DeleteReferral ...
func DeleteReferral(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Delete Referral")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")

	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferral, err := dsRefC.GetReferral(ctx, referralID)
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
	dsReferral.IsDirty = true
	dsReferral.ModifiedOn = time.Now()

	err = dsRefC.CreateReferral(ctx, *dsReferral)
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
		constants.RESPONSE_JSON_DATA:   nil,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// UploadDocuments ....
func UploadDocuments(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Update Referral Documents")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")
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

	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferral, err := dsRefC.GetReferral(ctx, referralID)
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
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
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
	toClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, dsReferral.ToPlaceID)
	if dsReferral.ToAddressID == "" && toClinic != nil {
		if err == nil && toClinic.AddressID != "" {
			dsReferral.ToPlaceID = toClinic.PlaceID
			dsReferral.ToClinicName = toClinic.Name
			dsReferral.ToClinicAddress = toClinic.Address
			dsReferral.ToAddressID = toClinic.AddressID
			dsReferral.ToEmail = toClinic.EmailAddress
			dsReferral.ToClinicPhone = toClinic.PhoneNumber
		}
	}
	var commentReasons contracts.Comment
	currentID, _ := uuid.NewUUID()
	commentReasons.MessageID = currentID.String()
	commentReasons.Channel = "c2c"
	commentReasons.Text = "Document List"
	commentReasons.UserID = userEmail
	if err == nil && toClinic != nil {
		zone, _ := tz.GetZone(tz.Point{
			Lon: toClinic.Location.Long, Lat: toClinic.Location.Lat,
		})

		location, _ := time.LoadLocation(zone[0])
		dsReferral.ModifiedOn = time.Now().In(location)
		commentReasons.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)

	} else {
		dsReferral.ModifiedOn = time.Now()
		commentReasons.TimeStamp = time.Now().UTC().UnixNano() / int64(time.Millisecond)
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
	docIDNames := make([]string, 0)
	docsMedia := make([]contracts.Media, 0)

	// Stage 2 Upload files from
	// parse request
	const _24K = 256 << 20
	if err = c.Request.ParseMultipartForm(_24K); err == nil {
		for _, fheaders := range c.Request.MultipartForm.File {
			for _, hdr := range fheaders {
				// open uploaded
				var infile multipart.File
				if infile, err = hdr.Open(); err != nil {

					c.AbortWithStatusJSON(
						http.StatusBadRequest,
						gin.H{
							constants.RESPONSE_JSON_DATA:   nil,
							constants.RESPONSDE_JSON_ERROR: fmt.Errorf("Bad files sent to backend"),
						},
					)
					return
				}
				fileName := hdr.Filename
				reader, err := storageC.DownloadSingleFile(ctx, referralID, constants.SD_REFERRAL_BUCKET, fileName)
				if err == nil && reader != nil {
					timeNow := time.Now()
					stripFile := strings.Split(fileName, ".")
					name := stripFile[0]
					name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
					fileName = name + "." + stripFile[1]
				}
				bucketPath := referralID + "/" + fileName
				buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
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
				imageBuffer := bytes.NewBuffer(nil)
				if _, err := io.Copy(imageBuffer, infile); err != nil {
					c.AbortWithStatusJSON(
						http.StatusInternalServerError,
						gin.H{
							constants.RESPONSE_JSON_DATA:   nil,
							constants.RESPONSDE_JSON_ERROR: err.Error(),
						},
					)
					return
				}
				currentBytes := imageBuffer.Bytes()
				_, err = io.Copy(buckerW, bytes.NewReader(currentBytes))
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
				buckerW.Close()
				extentionFile := strings.Split(fileName, ".")
				ext := extentionFile[len(extentionFile)-1]
				var docMedia contracts.Media
				docMedia.Name = fileName
				docMedia.Image = ""
				switch strings.ToLower(ext) {
				case "jpg", "jpeg":
					img, err := jpeg.Decode(bytes.NewReader(currentBytes))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				case "png":
					img, err := png.Decode(bytes.NewReader(currentBytes))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				}
				docsMedia = append(docsMedia, docMedia)
				docIDNames = append(docIDNames, fileName)
			}
		}
		err = storageC.ZipFile(ctx, referralID, constants.SD_REFERRAL_BUCKET)
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
	}

	dsReferral.Documents = append(dsReferral.Documents, docIDNames...)
	err = dsRefC.CreateReferral(ctx, *dsReferral)
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
	var refComments contracts.ReferralComments
	//commentReasons.Files = docIDNames
	commentReasons.Media = docsMedia
	refComments.Comments = append(refComments.Comments, commentReasons)
	_, err = ProcessComments(ctx, gproject, dsReferral.ReferralID, refComments)
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
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// DownloadDocumentsAsZip .....
func DownloadDocumentsAsZip(c *gin.Context) {
	log.Infof("Download Referral Documents")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	zipReader, err := storageC.DownloadAsZip(ctx, referralID, constants.SD_REFERRAL_BUCKET)
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
	fileNameDefault := referralID + ".zip"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileNameDefault))
	c.Header("Content-Type", "application/zip")

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

// DownloadSingleFile .....
func DownloadSingleFile(c *gin.Context) {
	log.Infof("Download Referral Documents")
	ctx := c.Request.Context()
	referralID := c.Param("referralId")
	fileName := c.Query("fileName")
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	fileReader, err := storageC.DownloadSingleFile(ctx, referralID, constants.SD_REFERRAL_BUCKET, fileName)
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
	fileNameDefault := fileName
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileNameDefault))
	c.Header("Content-Type", "application/zip")

	if _, err := io.Copy(c.Writer, fileReader); err != nil {
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

// GetAllReferralsGD ....
func GetAllReferralsGD(c *gin.Context) {
	log.Infof("Get all referrals")

	addressID := c.Query("addressId")
	ctx := c.Request.Context()
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
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
	currentClinic, err := clinicDB.GetSingleClinic(ctx, addressID)
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
	dsReferrals := make([]contracts.DSReferral, 0)
	dsReferrals, err = dsRefC.GetAllReferralsGD(ctx, addressID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusNotFound,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	treatmentSummary, err := dsRefC.GetAllTreamentSummaryGD(ctx, currentClinic.PlaceID)
	mapGDStuff := make(map[string]contracts.DSReferral)
	for _, ref := range dsReferrals {
		mapGDStuff[ref.ReferralID] = ref
	}
	if err == nil && treatmentSummary != nil && len(treatmentSummary) > 0 {
		for _, ref := range treatmentSummary {
			mapGDStuff[ref.ReferralID] = ref
		}
		dsReferrals = make([]contracts.DSReferral, 0)
		for _, ref := range mapGDStuff {
			dsReferrals = append(dsReferrals, ref)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferrals,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetAllReferralsSP ....
func GetAllReferralsSP(c *gin.Context) {
	log.Infof("Get all referrals")

	addressID := c.Query("placeId")
	ctx := c.Request.Context()
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
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
	currentClinic, err := clinicDB.GetSingleClinic(ctx, addressID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusNotFound,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferrals, err := dsRefC.GetAllReferralsSP(ctx, currentClinic.PlaceID, currentClinic.Name)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusNotFound,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferrals,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// GetOneReferral ....
func GetOneReferral(c *gin.Context) {
	log.Infof("Get all referrals")

	referralID := c.Param("referralId")
	ctx := c.Request.Context()
	_, _, gproject, err := getUserDetails(ctx, c.Request)
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
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferral, err := dsRefC.GetReferral(ctx, referralID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusNotFound,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// ReceiveReferralMail ...
func ReceiveReferralMail(c *gin.Context) {
	log.Infof("Referral Email Receieved")
	const _24K = 256 << 20
	err := c.Request.ParseMultipartForm(_24K)
	emails := c.Request.MultipartForm.Value["email"][0]
	log.Errorf(emails)
	parsedEmail, err := pe.Parse(strings.NewReader(emails))
	fromEmail := parsedEmail.From[0].Address
	subject := parsedEmail.Subject
	ctx := c.Request.Context()
	gproject := googleprojectlib.GetGoogleProjectID()
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
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
	dsReferral, err := dsRefC.GetReferral(ctx, subject)
	if err != nil {
		dsReferralAll, err := dsRefC.GetReferralFromEmail(ctx, fromEmail)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
			return
		}
		dsReferral = &dsReferralAll[0]
	}
	currentBody := parsedEmail.TextBody
	currentComments := make([]contracts.Comment, 0)
	docIDNames := make([]string, 0)
	docsMedia := make([]contracts.Media, 0)

	// Stage 2 Upload files from
	// parse request
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
	for _, attach := range parsedEmail.Attachments {
		fileName := attach.Filename
		reader, err := storageC.DownloadSingleFile(ctx, dsReferral.ReferralID, constants.SD_REFERRAL_BUCKET, fileName)
		if err == nil && reader != nil {
			timeNow := time.Now()
			stripFile := strings.Split(fileName, ".")
			name := stripFile[0]
			name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
			fileName = name + "." + stripFile[1]
		}
		bucketPath := dsReferral.ReferralID + "/" + fileName
		buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
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
		imageBuffer := bytes.NewBuffer(nil)
		if _, err := io.Copy(imageBuffer, attach.Data); err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())

		}
		currentBytes := imageBuffer.Bytes()
		_, err = io.Copy(buckerW, bytes.NewReader(currentBytes))
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
		buckerW.Close()
		extentionFile := strings.Split(fileName, ".")
		ext := extentionFile[len(extentionFile)-1]
		var docMedia contracts.Media
		docMedia.Name = fileName
		docMedia.Image = ""
		switch strings.ToLower(ext) {
		case "jpg", "jpeg":
			img, err := jpeg.Decode(bytes.NewReader(currentBytes))
			if err != nil {
				resized := resize.Resize(1280, 720, img, resize.Lanczos3)
				buf := new(bytes.Buffer)
				err := jpeg.Encode(buf, resized, nil)
				if err != nil {
					docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
				}
			}
		case "png":
			img, err := png.Decode(bytes.NewReader(currentBytes))
			if err != nil {
				resized := resize.Resize(1280, 720, img, resize.Lanczos3)
				buf := new(bytes.Buffer)
				err := jpeg.Encode(buf, resized, nil)
				if err != nil {
					docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
				}
			}
		}
		docsMedia = append(docsMedia, docMedia)
		docIDNames = append(docIDNames, fileName)
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
	currentDS := dsReferral.ToPlaceID
	latLong := contracts.ClinicLocation{}
	getClinic, err := clinicMetaDB.GetSingleClinicViaPlace(ctx, currentDS)
	if err == nil && getClinic.PhysicalClinicsRegistration.Name != "" {
		latLong = getClinic.Location
	} else {
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
		details, _ := mapClient.FindPlaceFromID(currentDS)
		latLong.Lat = details.Geometry.Location.Lat
		latLong.Long = details.Geometry.Location.Lng
	}
	zone, err := tz.GetZone(tz.Point{
		Lon: latLong.Long, Lat: latLong.Lat,
	})
	location, _ := time.LoadLocation(zone[0])

	if len(docIDNames) > 0 {
		var uploadComment contracts.Comment
		uploadComment.Channel = contracts.SPCBox
		if dsReferral.PatientEmail != "" {
			uploadComment.UserID = dsReferral.PatientEmail
		} else {
			uploadComment.UserID = dsReferral.PatientPhone
		}
		id, _ := uuid.NewUUID()
		uploadComment.MessageID = id.String()
		uploadComment.Media = docsMedia
		uploadComment.Text = "New documents are uploaded by " + dsReferral.PatientFirstName + " " + dsReferral.PatientLastName
		uploadComment.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
		currentComments = append(currentComments, uploadComment)
		err = storageC.ZipFile(ctx, dsReferral.ReferralID, constants.SD_REFERRAL_BUCKET)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
	}

	dsReferral.Documents = append(dsReferral.Documents, docIDNames...)

	var comm contracts.Comment
	id, _ := uuid.NewUUID()
	comm.MessageID = id.String()
	comm.Channel = contracts.SPCBox
	if dsReferral.PatientEmail != "" {
		comm.UserID = dsReferral.PatientEmail
	} else {
		comm.UserID = dsReferral.PatientPhone
	}
	comm.Text = currentBody
	comm.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
	currentComments = append(currentComments, comm)

	err = dsRefC.CreateMessage(ctx, *dsReferral, currentComments)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
	dsReferral.ModifiedOn = time.Now().In(location)

	err = dsRefC.CreateReferral(ctx, *dsReferral)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
	if err != nil {
		log.Errorf("Error processing sms error:%v ", err.Error())
	}
	if dsReferral.ToEmail != "" {
		sgClient.SendClinicNotification(dsReferral.ToEmail, dsReferral.ToClinicName,
			dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)

	} else {
		sgClient.SendClinicNotification(constants.SD_ADMIN_EMAIL, dsReferral.ToClinicName,
			dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)
	}
}

// ReceiveAutoSummaryMail ...
func ReceiveAutoSummaryMail(c *gin.Context) {
	log.Infof("Referral Email Receieved")
	const _24K = 256 << 20
	err := c.Request.ParseMultipartForm(_24K)
	emails := c.Request.MultipartForm.Value["email"][0]
	log.Errorf(emails)
	parsedEmail, err := pe.Parse(strings.NewReader(emails))
	fromEmail := parsedEmail.From[0].Address
	toEmail := parsedEmail.To[0].Address
	subject := parsedEmail.Subject
	ctx := c.Request.Context()
	gproject := googleprojectlib.GetGoogleProjectID()
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("No clinics found for incoming email: %v", err.Error())
		return
	}
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("No clinics found for incoming email: %v", err.Error())
		return
	}
	domainName := strings.Split(fromEmail, "@")[1]
	var dsReferral contracts.DSReferral
	dsReferral.IsSummary = true
	currentClinicInbound, err := clinicDB.GetAllClinicsByEmail(ctx, fromEmail)
	if err == nil && len(currentClinicInbound) > 0 {
		oneClinicFrom := currentClinicInbound[0]
		dsReferral.ToAddressID = oneClinicFrom.AddressID
		dsReferral.ToPlaceID = oneClinicFrom.PlaceID
		dsReferral.ToClinicAddress = oneClinicFrom.Address
		dsReferral.ToClinicName = oneClinicFrom.Name
		dsReferral.ToClinicPhone = oneClinicFrom.PhoneNumber
		dsReferral.ToEmail = oneClinicFrom.EmailAddress
	} else {
		currentClinicInbound, err = clinicDB.GetAllClinicsByDomain(ctx, domainName)
		if err != nil && len(currentClinicInbound) > 0 {
			oneClinicFrom := currentClinicInbound[0]
			dsReferral.ToAddressID = oneClinicFrom.AddressID
			dsReferral.ToPlaceID = oneClinicFrom.PlaceID
			dsReferral.ToClinicAddress = oneClinicFrom.Address
			dsReferral.ToClinicName = oneClinicFrom.Name
			dsReferral.ToClinicPhone = oneClinicFrom.PhoneNumber
			dsReferral.ToEmail = oneClinicFrom.EmailAddress
		} else {
			log.Errorf("No clinics inbound found for incoming email: %v", err.Error())
			return
		}
	}
	currentClinicOutBoud, err := clinicDB.GetAllClinicsByAutoEmail(ctx, toEmail)
	if err != nil {
		log.Errorf("No clinics outbound found for incoming email: %v", err.Error())
		return
	}
	toClinic := currentClinicOutBoud[0]
	dsReferral.FromAddressID = toClinic.AddressID
	dsReferral.FromEmail = toClinic.EmailAddress
	dsReferral.FromPlaceID = toClinic.PlaceID
	dsReferral.FromClinicAddress = toClinic.Address
	dsReferral.FromClinicName = toClinic.Name
	dsReferral.FromClinicPhone = toClinic.PhoneNumber
	currentRefUUID, _ := uuid.NewUUID()
	uniqueRefID := currentRefUUID.String()
	dsReferral.ReferralID = uniqueRefID
	currentBody := parsedEmail.TextBody
	currentComments := make([]contracts.Comment, 0)
	docIDNames := make([]string, 0)
	docsMedia := make([]contracts.Media, 0)
	// Stage 2 Upload files from
	// parse request
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
	foundOne := false
	ocrText := ""
	var res *docconv.Response
	patientFirstName := ""
	patientLastName := ""
	for _, attch := range parsedEmail.Attachments {
		fileName := attch.Filename
		reader, err := storageC.DownloadSingleFile(ctx, dsReferral.ReferralID, constants.SD_REFERRAL_BUCKET, fileName)
		if err == nil && reader != nil {
			timeNow := time.Now()
			stripFile := strings.Split(fileName, ".")
			name := stripFile[0]
			name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
			fileName = name + "." + stripFile[1]
		}
		bucketPath := dsReferral.ReferralID + "/" + fileName
		saveFileReader, _ := ioutil.ReadAll(attch.Data)
		buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
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
		if !foundOne || patientFirstName == "" {
			foundOne = true
			res, err = docconv.Convert(bytes.NewReader(saveFileReader), "application/pdf", true)
			if err != nil {
				log.Errorf("deconv error: %v", err.Error())
			}
			if res != nil {
				ocrText = res.Body
			}
			patientIndex := -1
			wordFields := strings.Fields(ocrText)
			for i, word := range wordFields {
				if strings.ToLower(word) == "patient" {
					patientIndex = i
					break
				}
			}
			if patientIndex >= 0 {
				patientFirstName = strings.Title(strings.ToLower(wordFields[patientIndex+1]))
				patientLastName = strings.Title(strings.ToLower(wordFields[patientIndex+2]))
			}

		}
		_, err = io.Copy(buckerW, bytes.NewReader(saveFileReader))
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
		buckerW.Close()
		docIDNames = append(docIDNames, fileName)
	}

	if patientFirstName != "" {
		dsReferral.PatientFirstName = patientFirstName
		dsReferral.PatientLastName = patientLastName

	} else {
		dsReferral.PatientFirstName = "Treament"
		dsReferral.PatientLastName = "Summary"
	}
	log.Infof("PatientName: %s", dsReferral.PatientFirstName+dsReferral.PatientLastName)
	// try to find existing patient summary
	var existingReferralMain *contracts.DSReferral
	var existingSummary *contracts.DSReferral

	if dsReferral.PatientFirstName != "Treament" {
		existingReferrals, err := dsRefC.GetReferralUsingFields(ctx, dsReferral.FromEmail, dsReferral.PatientFirstName, dsReferral.PatientLastName)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		} else {
			for _, ref := range existingReferrals {
				if !ref.IsSummary {
					existingReferralMain = &ref
				} else {
					existingSummary = &ref
				}
			}
		}
		if existingReferralMain != nil {
			for _, attch := range parsedEmail.Attachments {
				fileName := attch.Filename
				reader, err := storageC.DownloadSingleFile(ctx, existingReferralMain.ReferralID, constants.SD_REFERRAL_BUCKET, fileName)
				if err == nil && reader != nil {
					timeNow := time.Now()
					stripFile := strings.Split(fileName, ".")
					name := stripFile[0]
					name += strconv.Itoa(timeNow.Year()) + timeNow.Month().String() + strconv.Itoa(timeNow.Day()) + strconv.Itoa(timeNow.Second())
					fileName = name + "." + stripFile[1]
				}
				bucketPath := existingReferralMain.ReferralID + "/" + fileName
				saveFileReader, _ := ioutil.ReadAll(attch.Data)
				buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
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
				_, err = io.Copy(buckerW, bytes.NewReader(saveFileReader))
				if err != nil {
					log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
				}
				buckerW.Close()
				extentionFile := strings.Split(fileName, ".")
				ext := extentionFile[len(extentionFile)-1]
				var docMedia contracts.Media
				docMedia.Name = fileName
				docMedia.Image = ""
				switch strings.ToLower(ext) {
				case "jpg", "jpeg":
					img, err := jpeg.Decode(bytes.NewReader(saveFileReader))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				case "png":
					img, err := png.Decode(bytes.NewReader(saveFileReader))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				}
				docsMedia = append(docsMedia, docMedia)
				docIDNames = append(docIDNames, fileName)
			}
			if len(docIDNames) > 0 {
				existingReferralMain.Documents = append(existingReferralMain.Documents, docIDNames...)
				err = storageC.ZipFile(ctx, existingReferralMain.ReferralID, constants.SD_REFERRAL_BUCKET)
				if err != nil {
					log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
				}
			}
		}
	}
	if existingSummary != nil {
		dsReferral = *existingSummary
	}
	dsReferral.IsDirty = false
	dsReferral.IsNew = false
	dsReferral.SummaryText = ocrText
	dsReferral.Status.GDStatus = "completed"
	dsReferral.Status.SPStatus = "completed"
	zone, err := tz.GetZone(tz.Point{
		Lon: toClinic.Location.Long, Lat: toClinic.Location.Lat,
	})
	location, _ := time.LoadLocation(zone[0])
	dsReferral.CreatedOn = time.Now().In(location)
	dsReferral.ModifiedOn = time.Now().In(location)
	if len(docIDNames) > 0 {
		var uploadComment contracts.Comment
		uploadComment.Channel = contracts.GDCBox
		if dsReferral.PatientEmail != "" {
			uploadComment.UserID = dsReferral.PatientEmail
		} else {
			uploadComment.UserID = dsReferral.PatientPhone
		}
		id, _ := uuid.NewUUID()
		uploadComment.MessageID = id.String()
		uploadComment.Media = docsMedia
		uploadComment.Text = "New documents are uploaded by " + dsReferral.ToClinicName
		uploadComment.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
		currentComments = append(currentComments, uploadComment)
		err = storageC.ZipFile(ctx, dsReferral.ReferralID, constants.SD_REFERRAL_BUCKET)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
	}

	dsReferral.Documents = append(dsReferral.Documents, docIDNames...)
	var comm contracts.Comment
	id, _ := uuid.NewUUID()
	comm.MessageID = id.String()
	comm.Channel = contracts.GDCBox
	comm.UserID = dsReferral.ToEmail
	comm.Text = currentBody
	comm.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
	currentComments = append(currentComments, comm)

	err = dsRefC.CreateMessage(ctx, dsReferral, currentComments)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
	if existingReferralMain != nil {
		err = dsRefC.CreateMessage(ctx, *existingReferralMain, currentComments)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
		existingReferralMain.ModifiedOn = time.Now()
		err = dsRefC.CreateReferral(ctx, *existingReferralMain)
	}
	dsReferral.ModifiedOn = time.Now()

	err = dsRefC.CreateReferral(ctx, dsReferral)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
	if err != nil {
		log.Errorf("Error processing sms error:%v ", err.Error())
	}
	sendPatientComments := make([]string, 0)
	for _, comment := range currentComments {
		if comment.Channel == contracts.GDCBox && dsReferral.ToEmail != "" && comment.UserID == dsReferral.ToEmail {
			sendPatientComments = append(sendPatientComments, comment.Text)
		}
	}
	y, m, d := dsReferral.ModifiedOn.Date()
	dateString := fmt.Sprintf("%d-%d-%d", y, int(m), d)
	if dsReferral.FromEmail != "" {

		sgClient.SendAutoEmailNotificationToGD(dsReferral.FromEmail, dsReferral.FromClinicName,
			dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName, dsReferral.PatientPhone, dsReferral.ReferralID, dateString, sendPatientComments)

	} else {

		sgClient.SendAutoEmailNotificationToGD(constants.SD_ADMIN_EMAIL, dsReferral.FromClinicName,
			dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName, dsReferral.PatientPhone, dsReferral.ReferralID, dateString, sendPatientComments)

	}
}

// ScheduleDemo ...
func ScheduleDemo(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindWith(&data, binding.JSON); err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				constants.RESPONSE_JSON_DATA:   nil,
				constants.RESPONSDE_JSON_ERROR: err.Error(),
			},
		)
		return
	}
	sgClient := sendgrid.NewSendGridClient()
	err := sgClient.InitializeSendGridClient()
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

	sgClient.SendLiveDemoRequest(data)
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   "scheduled",
		constants.RESPONSDE_JSON_ERROR: nil,
	})
	return
}

// TextRecievedPatient ...
func TextRecievedPatient(c *gin.Context) {
	log.Infof("Processing incoming text")
	err := c.Request.ParseForm()
	if err != nil {
		log.Errorf("Error parsing text recieve 1: %v", err.Error())
	}
	ctx := c.Request.Context()
	clientSMS := sms.NewSMSClient()
	err = clientSMS.InitializeSMSClient()
	if err != nil {
		log.Errorf("Error parsing text recieve 2: %v", err.Error())
	}
	form := c.Request.Form
	incomingPhone := form["From"][0]
	receivingCustomPhone := form["To"][0]
	receivingCustomPhone = strings.Replace(receivingCustomPhone, "-", "", -1)
	receivingCustomPhone = strings.Replace(receivingCustomPhone, "(", "", -1)
	receivingCustomPhone = strings.Replace(receivingCustomPhone, ")", "", -1)
	receivingCustomPhone = strings.Replace(receivingCustomPhone, " ", "", -1)
	incomingText := ""
	if text, ok := form["Body"]; ok {
		incomingText = text[0]
	}
	filePatients := make(map[string]*io.ReadCloser)
	for key, formValue := range form {
		if strings.Contains(strings.ToLower(key), "mediaurl") {
			currentURL := formValue[0]
			fileName, reader, err := clientSMS.GetMedia(ctx, currentURL)
			if err != nil {
				log.Errorf("Error parsing text recieve 3: %v", err.Error())
				continue
			}
			filePatients[fileName] = reader
		}
	}
	gproject := googleprojectlib.GetGoogleProjectID()
	if err != nil {
		log.Errorf("Error parsing text recieve 4: %v", err.Error())
	}
	dsRefC := datastoredb.NewReferralHandler()
	err = dsRefC.InitializeDataBase(ctx, gproject)
	if err != nil {
		log.Errorf("Error parsing text recieve: %v", err.Error())
	}
	dsReferrals, err := dsRefC.ReferralFromPatientPhone(ctx, incomingPhone)
	if err != nil || len(dsReferrals) <= 0 {
		log.Errorf("Referral not gound: %v", err.Error())
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
	currentDS := dsReferrals[0].ToPlaceID
	latLong := contracts.ClinicLocation{}
	getClinic, err := clinicMetaDB.GetSingleClinicViaPlace(ctx, currentDS)
	if err == nil && getClinic.PhysicalClinicsRegistration.Name != "" {
		latLong = getClinic.Location
	} else {
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
		details, _ := mapClient.FindPlaceFromID(currentDS)
		latLong.Lat = details.Geometry.Location.Lat
		latLong.Long = details.Geometry.Location.Lng
	}
	zone, err := tz.GetZone(tz.Point{
		Lon: latLong.Long, Lat: latLong.Lat,
	})
	location, _ := time.LoadLocation(zone[0])
	for _, dsReferral := range dsReferrals {
		if dsReferral.CommunicationPhone != "" && dsReferral.CommunicationPhone != receivingCustomPhone {
			continue
		}
		if incomingText != "" {
			var commText contracts.Comment
			commText.UserID = dsReferral.PatientEmail
			commText.Channel = contracts.SPCBox
			commText.Text = incomingText
			commText.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
			id, _ := uuid.NewUUID()
			commText.MessageID = id.String()
			err = dsRefC.CreateMessage(ctx, dsReferral, []contracts.Comment{commText})
			if err != nil {
				log.Errorf("Error processing sms error:%v ", err.Error())
			}
		}
		docIDNames := make([]string, 0)
		docsMedia := make([]contracts.Media, 0)

		var commText contracts.Comment

		if len(filePatients) > 0 {
			commText.Channel = contracts.SPCBox
			commText.Text = "New documents uploaded by " + dsReferral.PatientFirstName + " " + dsReferral.PatientLastName
			commText.TimeStamp = time.Now().In(location).UTC().UnixNano() / int64(time.Millisecond)
			id, _ := uuid.NewUUID()
			commText.MessageID = id.String()
			if dsReferral.PatientEmail != "" {
				commText.UserID = dsReferral.PatientEmail
			} else {
				commText.UserID = dsReferral.PatientPhone
			}
			storageC := storage.NewStorageHandler()
			err = storageC.InitializeStorageClient(ctx, gproject)
			if err != nil {
				log.Errorf("Referral not gound: %v", err.Error())
			}
			var counter int64
			for fileName, fileBytes := range filePatients {
				currentBytes, err := ioutil.ReadAll(*fileBytes)
				counter++
				extension := strings.Split(fileName, ".")[1]
				fileName = dsReferral.PatientFirstName + strconv.Itoa(int(time.Now().UTC().Unix()+counter)) + "." + extension
				bucketPath := dsReferral.ReferralID + "/" + fileName
				buckerW, err := storageC.UploadToGCS(ctx, bucketPath)
				if err != nil {
					log.Errorf("Error processing uploading text error:%v ", err.Error())
				}
				_, err = io.Copy(buckerW, bytes.NewReader(currentBytes))
				if err != nil {
					log.Errorf("Error processing sms error:%v ", err.Error())
				}
				buckerW.Close()
				extentionFile := strings.Split(fileName, ".")
				ext := extentionFile[len(extentionFile)-1]
				var docMedia contracts.Media
				docMedia.Name = fileName
				docMedia.Image = ""
				switch strings.ToLower(ext) {
				case "jpg", "jpeg":
					img, err := jpeg.Decode(bytes.NewReader(currentBytes))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				case "png":
					img, err := png.Decode(bytes.NewReader(currentBytes))
					if err != nil {
						resized := resize.Resize(1280, 720, img, resize.Lanczos3)
						buf := new(bytes.Buffer)
						err := jpeg.Encode(buf, resized, nil)
						if err != nil {
							docMedia.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
						}
					}
				}
				docsMedia = append(docsMedia, docMedia)
				(*fileBytes).Close()
				docIDNames = append(docIDNames, fileName)
			}
			err = storageC.ZipFile(ctx, dsReferral.ReferralID, constants.SD_REFERRAL_BUCKET)
			if err != nil {
				log.Errorf("Error processing zipping text error:%v ", err.Error())
			}
		}
		if len(docIDNames) > 0 {
			commText.Media = docsMedia
			err = dsRefC.CreateMessage(ctx, dsReferral, []contracts.Comment{commText})
			if err != nil {
				log.Errorf("Error processing sms error:%v ", err.Error())
			}
		}
		dsReferral.ModifiedOn = time.Now().In(location)
		dsReferral.Documents = append(dsReferral.Documents, docIDNames...)
		err = dsRefC.CreateReferral(ctx, dsReferral)
		if err != nil {
			log.Errorf("Error processing sms error:%v ", err.Error())
		}
		sgClient := sendgrid.NewSendGridClient()
		err = sgClient.InitializeSendGridClient()
		if err != nil {
			log.Errorf("Error processing sms error:%v ", err.Error())
		}
		if dsReferral.ToEmail != "" {
			sgClient.SendClinicNotification(dsReferral.ToEmail, dsReferral.ToClinicName,
				dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)

		} else {
			sgClient.SendClinicNotification(constants.SD_ADMIN_EMAIL, dsReferral.ToClinicName,
				dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)
		}
	}
}

// Parse ..... ..
func Parse(request *http.Request) *contracts.ParsedEmail {
	result := contracts.ParsedEmail{
		Headers:     make(map[string]string),
		Body:        make(map[string]string),
		Attachments: make(map[string][]byte),
		RawRequest:  request,
	}
	result.Parse()
	return &result
}

// ProcessComments .....
func ProcessComments(ctx context.Context, gproject string, referralID string, referralDetails contracts.ReferralComments) ([]contracts.Comment, error) {
	dsRefC := datastoredb.NewReferralHandler()
	err := dsRefC.InitializeDataBase(ctx, gproject)
	if err != nil {
		return nil, err
	}
	dsReferral, err := dsRefC.GetReferral(ctx, referralID)
	if err != nil {
		return nil, err
	}
	clinicDB := datastoredb.NewClinicMetaHandler()
	err = clinicDB.InitializeDataBase(ctx, gproject)
	if err != nil {
		return nil, err
	}
	toClinic, err := clinicDB.GetSingleClinicViaPlace(ctx, dsReferral.ToPlaceID)
	if dsReferral.ToAddressID == "" || dsReferral.ToClinicPhone == "" {
		if err == nil && toClinic != nil && toClinic.AddressID != "" {
			dsReferral.ToPlaceID = toClinic.PlaceID
			dsReferral.ToClinicName = toClinic.Name
			dsReferral.ToClinicAddress = toClinic.Address
			dsReferral.ToEmail = toClinic.EmailAddress
			dsReferral.ToClinicPhone = toClinic.PhoneNumber
			dsReferral.ToAddressID = toClinic.AddressID
		}
	}
	if err == nil && toClinic != nil {
		zone, _ := tz.GetZone(tz.Point{
			Lon: toClinic.Location.Long, Lat: toClinic.Location.Lat,
		})

		location, _ := time.LoadLocation(zone[0])
		dsReferral.ModifiedOn = time.Now().In(location)

	} else {
		dsReferral.ModifiedOn = time.Now()
	}
	updatedComm := make([]contracts.Comment, 0)
	for _, comm := range referralDetails.Comments {
		currentID, _ := uuid.NewUUID()
		comm.MessageID = currentID.String()
		updatedComm = append(updatedComm, comm)

	}
	err = dsRefC.CreateMessage(ctx, *dsReferral, updatedComm)
	if err != nil {
		return nil, err
	}
	sgClient := sendgrid.NewSendGridClient()
	err = sgClient.InitializeSendGridClient()
	if err != nil {
		return nil, err
	}
	if dsReferral.IsNew {
		sendPatientComments := make([]string, 0)
		for _, newComm := range updatedComm {
			sendPatientComments = append(sendPatientComments, newComm.Text)
		}
		y, m, d := dsReferral.CreatedOn.Date()
		dateString := fmt.Sprintf("%d-%d-%d", y, int(m), d)
		if dsReferral.ToEmail != "" {
			err = sgClient.SendEmailNotificationSpecialist(dsReferral.ToEmail,
				dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName,
				dsReferral.PatientPhone, dsReferral.ReferralID, dateString, sendPatientComments)
		} else {
			err = sgClient.SendEmailNotificationSpecialist(constants.SD_ADMIN_EMAIL,
				dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName,
				dsReferral.PatientPhone, dsReferral.ReferralID, dateString, sendPatientComments)
		}
		if err != nil {
			if err != nil {
				log.Errorf("Failed to send email: %v", err.Error())
			}
		}
		if dsReferral.PatientEmail != "" {
			err = sgClient.SendEmailNotificationPatient(dsReferral.PatientEmail,
				dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ToClinicName,
				dsReferral.ToClinicPhone, dsReferral.ReferralID, dsReferral.ToClinicAddress, sendPatientComments)
			if err != nil {
				if err != nil {
					log.Errorf("Failed to send email: %v", err.Error())
				}
			}
		}
		clientSMS := sms.NewSMSClient()
		err = clientSMS.InitializeSMSClient()
		if err != nil {
			log.Errorf("Failed to send SMS: %v", err.Error())
		}
		if dsReferral.PatientPhone != "" {
			message1 := ""
			message1 = fmt.Sprintf(constants.PATIENT_MESSAGE, dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
				dsReferral.ToClinicName, dsReferral.ToClinicAddress, dsReferral.ToClinicPhone, sendPatientComments)
			fromPhone := ""
			if dsReferral.CommunicationPhone == "" {
				fromPhone = global.Options.ReferralPhone
			} else {
				fromPhone = dsReferral.CommunicationPhone
			}
			err = clientSMS.SendSMS(fromPhone, dsReferral.PatientPhone, message1)
			message2 := "Please submit your insurance information here to receive an accurate co-pay: "
			message2 += os.Getenv("SD_BASE_URL") + "/secure/insurance?referral=" + dsReferral.ReferralID
			err = clientSMS.SendSMS(fromPhone, dsReferral.PatientPhone, message2)

		}

	}
	wasNew := dsReferral.IsNew
	dsReferral.IsNew = false

	err = dsRefC.CreateReferral(ctx, *dsReferral)
	if err != nil {
		return nil, err
	}
	if !wasNew {
		for _, comm := range referralDetails.Comments {
			if comm.Channel == contracts.GDCBox {
				if dsReferral.ToEmail != "" && comm.UserID == dsReferral.FromEmail {
					sgClient.SendClinicNotification(dsReferral.ToEmail, dsReferral.ToClinicName,
						dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)

				} else if dsReferral.ToEmail != "" && comm.UserID == dsReferral.ToEmail {
					sgClient.SendClinicNotification(dsReferral.FromEmail, dsReferral.FromClinicName,
						dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)
				} else {
					sgClient.SendClinicNotification(constants.SD_ADMIN_EMAIL, dsReferral.ToClinicName,
						dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)
				}
			} else if comm.Channel == contracts.SPCBox {
				clientSMS := sms.NewSMSClient()
				err = clientSMS.InitializeSMSClient()
				if err != nil {
					return nil, err
				}
				if dsReferral.PatientPhone != "" {
					message1 := ""
					message1 = fmt.Sprintf(constants.PATIENT_MESSAGE_NOTICE, dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
						dsReferral.ToClinicName, comm.Text)
					fromPhone := ""
					if dsReferral.CommunicationPhone == "" {
						fromPhone = global.Options.ReferralPhone
					} else {
						fromPhone = dsReferral.CommunicationPhone
					}
					err = clientSMS.SendSMS(fromPhone, dsReferral.PatientPhone, message1)
				}
				if dsReferral.PatientEmail != "" {
					err = sgClient.SendCommentNotificationPatient(dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
						dsReferral.PatientEmail, comm.Text, dsReferral.ToClinicName, dsReferral.ReferralID)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return updatedComm, nil
}
