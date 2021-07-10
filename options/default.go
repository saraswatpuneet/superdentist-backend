package options

// Default contains the default option values. These are used to
// initialize the Options object especially for local development
var Default = []byte(`
{
	"debug": false,
	"rto": "referralsdev@mailer.superdentist.io",
	"pct": "d-789db895f1464d7ab85f3abd8ff14a90",
	"sct": "d-6d15be1d8baf4beb98775d233988c7fd",
	"gdc": "d-f64251df5dc84cc598cda2e7be98d18a",
	"gdcauto":"d-cd2ce581f5e14250a914d6a655d93f10",
	"cnn": "d-a4a7dc5e0bf0436cb7766a3631ee803d",
	"pnn": "d-7c54cb4262a64e10a344551c77a56ec9",
	"curi": "https://dev.superdentist.io",
	"refphone": "+17373772180",
	"dbhost":"34.70.1.167",
	"dbport": 5432,
	"dbname": "superdentistpg",
	"dbuser": "sdadmin",
	"dbpassword": "rpt6eq2xPcyLcxw7",
	"sslmode": "verify-ca",
	"sslRootCert": "./certs/server-ca-dev.pem",
	"sslCert": "./certs/client-cert-dev.pem",
	"sslKey": "./certs/client-key-dev.pem"
}
`)
