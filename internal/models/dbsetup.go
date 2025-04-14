package models

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
	logger   *zap.Logger
}

func NewDbCfg(host, user, pass, dbName string, log *zap.Logger) *DbCfg {
	return &DbCfg{
		host:     host,
		user:     user,
		password: pass,
		dbName:   dbName,
		logger:   log,
	}
}

func (c *DbCfg) dsn() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		c.host, c.user, c.password, c.dbName)
}

func NewDb(cfg *DbCfg) (*gorm.DB, error) {
	if dbConn != nil {
		cfg.logger.Info("dbConn connection already established, reusing connection")
		return dbConn, nil
	}
	cfg.logger.Info("initiating db connection")
	var err error
	dbConn, err = gorm.Open(postgres.Open(cfg.dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(dbConn); err != nil {
		return nil, err
	}
	cfg.logger.Info("db connection established")
	return dbConn, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Application{},
		&Ingress{},
		&ModSecProtectionConfig{},
	)
}

func db() *gorm.DB {
	if dbConn == nil {
		zap.S().Fatal("database connection not initialized, you must call NewDb(dbCfg) first")
	}
	return dbConn
}
