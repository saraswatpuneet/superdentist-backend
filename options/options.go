package options

import (
	"encoding/json"
	"fmt"
	"os"
)

// Options .. contains global options like ones read from environment variables
type Options struct {
	Debug                  bool   `json:"debug,omitempty"`
	DSName                 string `json:"dsname,omitempty"`
	MaxPayloadSize         int64  `json:"max_payload_size,omitempty"`
	MaxHeaderSize          int    `json:"max_header_size,omitempty"`
	ReplyTo                string `json:"rto,omitempty"`
	PatientConfTemp        string `json:"pct,omitempty"`
	SpecialistConfTemp     string `json:"sct,omitempty"`
	GDReferralComp         string `json:"gdc,omitempty"`
	ClinicNotificatioNew   string `json:"cnn,omitempty"`
	PatientNotificationNew string `json:"pnn,omitempty"`
	ContinueURL            string `json:"curi,omitempty"`
}

// New .. create a new instance
func New() *Options {
	return &Options{}
}

// InitOptions initializes the options
func InitOptions() (*Options, error) {
	options := New()
	if err := json.Unmarshal(Default, options); err != nil {
		return nil, fmt.Errorf("Options initialization unmarshal error: %v", err)
	}
	options.Debug = false
	options.DSName = os.Getenv("DS_NAMESPACE")
	options.ReplyTo = os.Getenv("SD_ADMIN_EMAIL_REPLYTO")
	options.PatientConfTemp = os.Getenv("SD_PATIENT_REF_CONF")
	options.SpecialistConfTemp = os.Getenv("SD_SPECIALIZT_REF_CONF")
	options.GDReferralComp = os.Getenv("GD_REFERRAL_COMPLETED")
	options.ClinicNotificatioNew = os.Getenv("CLINIC_NOTIFICATION_NEW")
	options.PatientNotificationNew = os.Getenv("PATINET_EMAIL_NOTIFICATION")
	options.ContinueURL = os.Getenv("CONTINUE_URL")
	return options, nil
}
