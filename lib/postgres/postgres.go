package postgres

import (
	"context"

	"github.com/jackc/pgx"
	_ "github.com/lib/pq"
	"github.com/superdentist/superdentist-backend/global"
)

//NewPostgresHandler return new database action
func NewPostgresHandler(ctx context.Context) error {
	pgxConfig := pgx.ConnConfig{
		Host:     global.Options.DBHost,
		Port:     uint16(global.Options.DBPort),
		User:     global.Options.DBUser,
		Password: global.Options.DBPassword,
		Database: global.Options.DBName,
	}
	pgxConnPoolConfig := pgx.ConnPoolConfig{
		ConnConfig:     pgxConfig,
		MaxConnections: 10,
	}
	global.PGXConfig = &pgxConfig
	connectionPool, err := pgx.NewConnPool(pgxConnPoolConfig)
	if err != nil {
		return err
	}
	global.PGXConn = connectionPool
	//rootCert := "sslrootcert=" + global.Options.RootCA + " "
	//sslKey := "sslkey=" + global.Options.SSLKey + " "
	//sslCert := "sslcert=" + global.Options.SSLCert + " "
	//connection := fmt.Sprintf("host=%s port=%v user=%s dbname=%s sslmode=%s password=%s", global.Options.DBHost, global.Options.DBPort, global.Options.DBUser, global.Options.DBName, global.Options.SSLMode, global.Options.DBPassword)
	//connection += " " + rootCert + sslKey + sslCert
	//db, err := sql.Open("postgres", connection)
	//if err != nil {
	//	return err
	//}
	//defer db.Close()
	//err = db.Ping()
	//if err != nil {
	//	return err
	//}
	return nil
}
