package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/superdentist/superdentist-backend/global"
)

// PGSDentist ....
type PGSDentist struct {
	ClientConn *sql.DB
}

//NewPostgresHandler return new database action
func NewPostgresHandler() (*PGSDentist, error) {
	rootCert := "sslrootcert=" + global.Options.RootCA + " "
	sslKey := "sslkey=" + global.Options.SSLKey + " "
	sslCert := "sslcert=" + global.Options.SSLCert + " "
	connection := fmt.Sprintf("host=%s port=%v user=%s dbname=%s sslmode=require password=%s", global.Options.DBHost, global.Options.DBPort, global.Options.DBUser, global.Options.DBName, global.Options.DBPassword)
	connection += " " + rootCert + sslKey + sslCert
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &PGSDentist{ClientConn: db}, nil
}
