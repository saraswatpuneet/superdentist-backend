package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	strip "github.com/grokify/html-strip-tags-go"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
	"github.com/superdentist/superdentist-backend/lib/googleprojectlib"
	"github.com/superdentist/superdentist-backend/lib/sendgrid"
	"github.com/superdentist/superdentist-backend/lib/storage"
)

// AddCommentsToReferral ...
func AddCommentsToReferral(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Add comments to Referral")
	ctx := c.Request.Context()
	referralID := c.Param("id")

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
	dsReferral.Comments = append(dsReferral.Comments, referralDetails.Comments...)
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
	for _, comm := range referralDetails.Comments {
		if comm.ChatBox == contracts.SPCBox {
			if dsReferral.ToEmail != "" {
				sgClient.SendClinicNotification(dsReferral.ToEmail, dsReferral.ToClinicName,
					dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)

			} else {
				sgClient.SendClinicNotification(constants.SD_ADMIN_EMAIL, dsReferral.ToClinicName,
					dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)
			}
		} else if comm.ChatBox == contracts.GDCBox {
			if dsReferral.FromEmail != "" {
				sgClient.SendClinicNotification(dsReferral.FromEmail, dsReferral.FromClinicName,
					dsReferral.PatientFirstName+" "+dsReferral.PatientLastName, dsReferral.ReferralID)

			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// UpdateReferralStatus ...
func UpdateReferralStatus(c *gin.Context) {
	// Stage 1  Load the incoming request
	log.Infof("Update Referral Status")
	ctx := c.Request.Context()
	referralID := c.Param("id")

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
	dsReferral.Status = referralDetails.Status
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
		for _, comment := range dsReferral.Comments {
			if comment.ChatBox == contracts.GDCBox {
				sendPatientComments = append(sendPatientComments, comment.Comment)
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
	referralID := c.Param("id")

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
	referralID := c.Param("id")
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
	// Stage 2 Upload files from
	// parse request
	const _24K = (1 << 10) * 24
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
				_, err = io.Copy(buckerW, infile)
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
				docIDNames = append(docIDNames, hdr.Filename)
			}
		}
		err = storageC.ZipFile(ctx, referralID)
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
		constants.RESPONSE_JSON_DATA:   dsReferral,
		constants.RESPONSDE_JSON_ERROR: nil,
	})
}

// DownloadDocumentsAsZip .....
func DownloadDocumentsAsZip(c *gin.Context) {
	log.Infof("Download Referral Documents")
	ctx := c.Request.Context()
	referralID := c.Param("id")
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
	zipReader, err := storageC.DownloadAsZip(ctx, referralID)
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
	dsReferrals, err := dsRefC.GetAllReferralsGD(ctx, addressID)
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

// GetAllReferralsSP ....
func GetAllReferralsSP(c *gin.Context) {
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
	dsReferrals, err := dsRefC.GetAllReferralsSP(ctx, addressID)
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

	referralID := c.Param("id")
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
	parsedEmail := Parse(c.Request)
	fromEmail := parsedEmail.Headers["From"]
	toEmail := parsedEmail.Headers["To"]
	subject := parsedEmail.Headers["Subject"]

	if toEmail != "referrals@mailer.superdentist.io" {
		log.Errorf("Email sent to bad actor" + " " + fromEmail + " " + subject)
	}
	bodyCleaned := make(map[string]string, 0)
	for key, text := range parsedEmail.Body {
		if strings.Contains(key, "html") {
			text = strings.ReplaceAll(text, ">", "> ")
			text = strip.StripTags(text)
			text = strings.ReplaceAll(text, "***Enter your message related to appointment,available date, questions etc.***", "")
			text = strings.ReplaceAll(text, "\n", "")
			text = strings.TrimSpace(text)
			bodyCleaned[key] = text
		}
	}
	parsedEmail.Body = bodyCleaned
	ctx := c.Request.Context()
	gproject := googleprojectlib.GetGoogleProjectID()
	dsRefC := datastoredb.NewReferralHandler()
	err := dsRefC.InitializeDataBase(ctx, gproject)
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
		dsReferral, err = dsRefC.GetReferralFromEmail(ctx, fromEmail)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
	}
	currentBody := parsedEmail.Body
	currentComments := make([]contracts.Comment, 0)
	docIDNames := make([]string, 0)
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
	for fileName, fileBytes := range parsedEmail.Attachments {
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
		_, err = io.Copy(buckerW, bytes.NewReader(fileBytes))
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
		buckerW.Close()
		docIDNames = append(docIDNames, fileName)
	}
	if len(docIDNames) > 0 {
		var uploadComment contracts.Comment
		uploadComment.ChatBox = contracts.PTCBOX
		uploadComment.Comment = "New documents are uploaded by " + dsReferral.PatientFirstName + " " + dsReferral.PatientLastName
		uploadComment.Time = time.Now().Unix()
		currentComments = append(currentComments, uploadComment)
	}
	err = storageC.ZipFile(ctx, dsReferral.ReferralID)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}

	dsReferral.Documents = append(dsReferral.Documents, docIDNames...)
	for _, text := range currentBody {
		var comm contracts.Comment
		comm.ChatBox = contracts.PTCBOX
		comm.Comment = text
		comm.Time = time.Now().Unix()
		currentComments = append(currentComments, comm)
	}
	dsReferral.Comments = append(dsReferral.Comments, currentComments...)
	dsReferral.ModifiedOn = time.Now()

	err = dsRefC.CreateReferral(ctx, *dsReferral)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
}

// TextRecievedPatient ...
func TextRecievedPatient(c *gin.Context) {
	log.Infof("Processing incoming text")
	smsDetails := make(map[string]interface{})
	for key, value := range c.Request.Header{
		log.Infof("key: %v", key)
		log.Infof("value: %v", value)

	}
	bytesSms, _:= ioutil.ReadAll(c.Request.Body)
	log.Infof("bytes: %v", string(bytesSms))
	if err := json.Unmarshal(bytesSms, &smsDetails); err != nil {
		log.Errorf("Bad sms received: %v", err.Error())
	}
	log.Infof("incoming sms: %v", smsDetails)

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
