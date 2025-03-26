package database

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbConn *gorm.DB

type DbCfg struct {
	host     string
	user     string
	password string
	dbName   string
}

func NewDbCfg(host, user, pass, dbName string) *DbCfg {
	return &DbCfg{
		host:     host,
		user:     user,
		password: pass,
		dbName:   dbName,
	}
}

func (c *DbCfg) dsn() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		c.host, c.user, c.password, c.dbName)
}

func NewDb(cfg *DbCfg) (*gorm.DB, error) {
	if dbConn != nil {
		zap.S().Info("dbConn connection already established, reusing connection")
		return dbConn, nil
	}
	zap.S().Info("initiating dbConn connection")
	db, err := gorm.Open(postgres.Open(cfg.dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	zap.S().Info("dbConn connection established")
	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Application{},
		&Ingress{},
	)
}

func db() *gorm.DB {
	if dbConn == nil {
		zap.S().Fatal("database connection not initialized, you must call NewDb(dbCfg) first")
	}
	return dbConn
}
