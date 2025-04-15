package models

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	dbConn *gorm.DB
	logger *zap.Logger
)

type DbCfg struct {
	host     string
	user     string
	password string
	dbName   string
}

func NewDbCfg(host, user, pass, dbName string, log *zap.Logger) *DbCfg {
	logger = log
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
		logger.Info("dbConn connection already established, reusing connection")
		return dbConn, nil
	}
	logger.Info("initiating db connection")
	var err error
	dbConn, err = gorm.Open(postgres.Open(cfg.dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(dbConn); err != nil {
		return nil, err
	}
	logger.Info("db connection established")
	return dbConn, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Application{},
		&Ingress{},
		&Protection{},
	)
}

func db() *gorm.DB {
	if dbConn == nil {
		zap.S().Fatal("database connection not initialized, you must call NewDb(dbCfg) first")
	}
	return dbConn
}

func mlog() *zap.Logger {
	if logger == nil {
		zap.S().Fatal("logger not initialized, you must call NewDb(dbCfg) first")
	}
	return logger
}
