package handlers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	strip "github.com/grokify/html-strip-tags-go"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/contracts"
	"github.com/superdentist/superdentist-backend/lib/datastoredb"
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
	updatedComm := make([]contracts.Comment, 0)
	for _, comm := range referralDetails.Comments {
		comm.TimeStamp = time.Now().Unix()
		currentID, _ := uuid.NewUUID()
		comm.MessageID = currentID.String()
		updatedComm = append(updatedComm, comm)

	}
	err = dsRefC.CreateMessage(ctx, *dsReferral, updatedComm)
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
		message := fmt.Sprintf(constants.PATIENT_MESSAGE, dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
			dsReferral.ToClinicName, dsReferral.ToClinicAddress, dsReferral.ToClinicPhone, sendPatientComments)
		err = clientSMS.SendSMS(constants.SD_REFERRAL_PHONE, dsReferral.PatientPhone, message)

	}
	wasNew := dsReferral.IsNew
	dsReferral.IsNew = false
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
					c.AbortWithStatusJSON(
						http.StatusInternalServerError,
						gin.H{
							constants.RESPONSE_JSON_DATA:   nil,
							constants.RESPONSDE_JSON_ERROR: err.Error(),
						},
					)
					return
				}
				message := fmt.Sprintf(constants.PATIENT_MESSAGE_NOTICE, dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
					dsReferral.ToClinicName, comm.Text)
				clientSMS.SendSMS(constants.SD_REFERRAL_PHONE, dsReferral.PatientPhone, message)
				if dsReferral.PatientEmail != "" {
					err = sgClient.SendCommentNotificationPatient(dsReferral.PatientFirstName+" "+dsReferral.PatientLastName,
						dsReferral.PatientEmail, comm.Text, dsReferral.ToClinicName, dsReferral.ReferralID)
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
			}
		}
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
	dsReferrals, err := dsRefC.GetAllReferralsSP(ctx, currentClinic.PlaceID)
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
	parsedEmail := Parse(c.Request)
	fromEmail := parsedEmail.Headers["From"]
	toEmail := parsedEmail.Headers["To"]
	subject := parsedEmail.Headers["Subject"]
	re := regexp.MustCompile(`\<.*?\>`)
	fromSub := re.FindAllString(fromEmail, -1)[0]
	fromEmail = strings.Trim(fromSub, "<")
	fromEmail = strings.Trim(fromEmail, ">")
	toSub := re.FindAllString(toEmail, -1)[0]
	toEmail = strings.Trim(toSub, "<")
	toEmail = strings.Trim(toEmail, ">")
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
			text = strings.Split(text, "SuperDentist Admin")[0]
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
		uploadComment.Channel = contracts.SPCBox
		uploadComment.UserID = dsReferral.PatientEmail
		id, _ := uuid.NewUUID()
		uploadComment.MessageID = id.String()
		uploadComment.Text = "New documents are uploaded by " + dsReferral.PatientFirstName + " " + dsReferral.PatientLastName
		uploadComment.TimeStamp = time.Now().Unix()
		currentComments = append(currentComments, uploadComment)
		err = storageC.ZipFile(ctx, dsReferral.ReferralID)
		if err != nil {
			log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
		}
	}

	dsReferral.Documents = append(dsReferral.Documents, docIDNames...)
	for _, text := range currentBody {
		var comm contracts.Comment
		id, _ := uuid.NewUUID()
		comm.MessageID = id.String()
		comm.Channel = contracts.SPCBox
		comm.UserID = dsReferral.PatientEmail
		comm.Text = text
		comm.TimeStamp = time.Now().Unix()
		currentComments = append(currentComments, comm)
	}
	err = dsRefC.CreateMessage(ctx, *dsReferral, currentComments)
	if err != nil {
		log.Errorf("Error processing email"+" "+fromEmail+" "+subject+" error:%v ", err.Error())
	}
	dsReferral.ModifiedOn = time.Now()

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
	dsReferral, err := dsRefC.ReferralFromPatientPhone(ctx, incomingPhone)
	if err != nil {
		log.Errorf("Referral not gound: %v", err.Error())
	}
	if incomingText != "" {
		var commText contracts.Comment
		commText.UserID = dsReferral.PatientEmail
		commText.Channel = contracts.SPCBox
		commText.Text = incomingText
		commText.TimeStamp = time.Now().Unix()
		id, _ := uuid.NewUUID()
		commText.MessageID = id.String()
		err = dsRefC.CreateMessage(ctx, *dsReferral, []contracts.Comment{commText})
		if err != nil {
			log.Errorf("Error processing sms error:%v ", err.Error())
		}
	}
	docIDNames := make([]string, 0)

	if len(filePatients) > 0 {
		var commText contracts.Comment
		commText.Channel = contracts.SPCBox
		commText.Text = "New documents uploaded by " + dsReferral.PatientFirstName + " " + dsReferral.PatientLastName
		commText.TimeStamp = time.Now().Unix()
		id, _ := uuid.NewUUID()
		commText.MessageID = id.String()
		commText.UserID = dsReferral.PatientEmail
		err = dsRefC.CreateMessage(ctx, *dsReferral, []contracts.Comment{commText})
		if err != nil {
			log.Errorf("Error processing sms error:%v ", err.Error())
		}
		storageC := storage.NewStorageHandler()
		err = storageC.InitializeStorageClient(ctx, gproject)
		if err != nil {
			log.Errorf("Referral not gound: %v", err.Error())
		}
		var counter int64
		for fileName, fileBytes := range filePatients {
			counter++
			extension := strings.Split(fileName, ".")[1]
			fileName = dsReferral.PatientFirstName + strconv.Itoa(int(time.Now().Unix()+counter)) + "." + extension
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
			_, err = io.Copy(buckerW, *fileBytes)
			if err != nil {
				log.Errorf("Error processing sms error:%v ", err.Error())
			}
			buckerW.Close()
			(*fileBytes).Close()
			docIDNames = append(docIDNames, fileName)
		}
	}
	dsReferral.ModifiedOn = time.Now()

	err = dsRefC.CreateReferral(ctx, *dsReferral)
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
