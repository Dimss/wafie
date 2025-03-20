package database

import (
	"fmt"
	"github.com/Dimss/cwaf/internal/database/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

func NewDb(cfg DbCfg) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	models := []interface{}{&model.Ingress{}, &model.Ingress2{}}
	if err := db.AutoMigrate(models); err != nil {
		return nil, err
	}
	return db, nil
}
