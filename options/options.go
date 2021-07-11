package options

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
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
	GDReferralAuto         string `json:"gdcauto,omitempty"`
	ClinicNotificatioNew   string `json:"cnn,omitempty"`
	PatientNotificationNew string `json:"pnn,omitempty"`
	ContinueURL            string `json:"curi,omitempty"`
	ReferralPhone          string `json:"refphone,omitempty"`
	EncryptionKeyQR        string `json:"encryptionkeyqr,omitempty"`
	GCMQR                  cipher.AEAD
	DBHost                 string `json:"dbhost,omitempty"`
	DBPort                 int    `json:"dbport,omitempty"`
	DBName                 string `json:"dbname,omitempty"`
	DBUser                 string `json:"dbuser,omitempty"`
	DBPassword             string `json:"dbpassword,omitempty"`
	SSLMode                string `json:"sslmode,omitempty"`
	RootCA                 string `json:"sslRootCert,omitempty"`
	SSLKey                 string `json:"sslKey,omitempty"`
	SSLCert                string `json:"sslCert,omitempty"`
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
	if !options.Debug {
		options.DSName  = os.Getenv("DS_NAMESPACE")
		options.ReplyTo = os.Getenv("SD_ADMIN_EMAIL_REPLYTO")
		options.PatientConfTemp = os.Getenv("SD_PATIENT_REF_CONF")
		options.SpecialistConfTemp = os.Getenv("SD_SPECIALIZT_REF_CONF")
		options.GDReferralComp = os.Getenv("GD_REFERRAL_COMPLETED")
		options.GDReferralAuto = os.Getenv("GD_REFERRAL_AUTO")
		options.ClinicNotificatioNew = os.Getenv("CLINIC_NOTIFICATION_NEW")
		options.PatientNotificationNew = os.Getenv("PATINET_EMAIL_NOTIFICATION")
		options.ContinueURL = os.Getenv("CONTINUE_URL")
		options.ReferralPhone = os.Getenv("SD_REFERRAL_PHONE")
		options.EncryptionKeyQR = os.Getenv("QR_ENC_KEY")
		if options.EncryptionKeyQR != "" {
			key, _ := base64.StdEncoding.DecodeString(options.EncryptionKeyQR)
			c, err := aes.NewCipher(key)
			options.GCMQR = nil
			if err == nil {
				gcm, err := cipher.NewGCM(c)
				if err != nil {
					options.GCMQR = nil
				}
				options.GCMQR = gcm
			}
		}
		dbHost := os.Getenv("DB_HOST")
		if dbHost!= "" {
			options.DBHost = dbHost
		}
		options.DBPort = 5432
		dbName := os.Getenv("DB_NAME")
		if dbName != "" {
			options.DBName = dbName
		}
		dbUser := os.Getenv("DB_USER")
		if dbUser != "" {
			options.DBUser = dbUser
		}
		dbPassword := os.Getenv("DB_PASSWORD")
		if dbPassword != "" {
			options.DBPassword = dbPassword
		}
		sslMode := os.Getenv("SSL_MODE")
		if sslMode != "" {
			options.SSLMode = sslMode
		}
		rootCA := os.Getenv("SSL_ROOT_CA")
		if rootCA != "" {
			options.RootCA = rootCA
		}
		sslKey := os.Getenv("SSL_KEY")
		if sslKey != "" {
			options.SSLKey = sslKey
		}
		sslCert := os.Getenv("SSL_CERT")
		if sslCert != "" {
			options.SSLCert = sslCert
		}

	}
	return options, nil
}
