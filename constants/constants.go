package constants

const (
	// ENV_DEBUG ..... for debugging serverity
	ENV_DEBUG                  string = "DEBUG"
	MAX_READ_TIMEOUT                  = 50 // secs
	MAX_WRITE_TIMEOUT                 = 50 // secs
	RESPONSE_JSON_DATA         string = "data"
	RESPONSDE_JSON_ERROR       string = "error"
	SD_REFERRAL_PHONE                 = "+12157745591"
	SD_REFERRAL_BUCKET                = "superdentist-referrals"
	SD_MAIN_EMAIL                     = "superdentist.admin@superdentist.io"
	SD_ADMIN_EMAIL                    = "referrals@superdentist.io"
	SD_ADMIN_EMAIL_REPLYTO            = "referrals@mailer.superdentist.io"
	SD_PATIENT_REF_CONF               = "d-0cb214d233c0499691bfba7a42689ac7"
	SD_SPECIALIZT_REF_CONF            = "d-e9288c40cc76436db70a32dc4dba6efa"
	GD_REFERRAL_COMPLETED             = "d-5223f0628162417591e27d9810460ebc"
	CLINIC_NOTIFICATION_NEW           = "d-7ab65eeeb1144af285bc31aa39fb6873"
	VERIFICATION_EMAIL_NEW            = "d-2785f85539db4bb7ab9a2f763dee89b9"
	PASSWORD_RESET_EMAIL              = "d-5f1d97747dd249fbb576c5daa543f430"
	PATINET_EMAIL_NOTIFICATION        = "d-c2e691190e1145d58d2fdda9782257ed"
	PATIENT_MESSAGE                   = `Hi %s 
	
	You've been referred to %s.
	
	Address: %s
	
	Phone: %s
	
	Comments: %s
	
	You can text directly in this thread to chat with your specialist and book a convenient appointment time. Your specialist may also message you first!`
	PATIENT_MESSAGE_NOTICE = ` Hi %s 
	
	Message from %s : %s`
)
