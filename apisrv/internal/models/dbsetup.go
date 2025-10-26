package models

import (
	"fmt"

	"github.com/Dimss/wafie/apisrv/internal/models/sql"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	dbConn   *gorm.DB
	logger   *zap.Logger
	migrated bool
	seeded   bool
	//	sql = `
	//CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	//
	//CREATE OR REPLACE FUNCTION array_compare_as_set(arr1 anyarray, arr2 anyarray) RETURNS boolean AS $$
	//SELECT CASE
	//           WHEN array_dims(arr1) <> array_dims(arr2) THEN 'f'
	//           WHEN array_length(arr1,1) <> array_length(arr2,1) THEN 'f'
	//           ELSE NOT EXISTS (
	//               SELECT 1
	//               FROM unnest(arr1) a
	//                        FULL JOIN unnest(arr2) b ON (a=b)
	//               WHERE a IS NULL or b IS NULL
	//           )
	//           END
	//$$ LANGUAGE SQL IMMUTABLE;
	//
	//CREATE OR REPLACE FUNCTION upstreams_update_trigger() RETURNS TRIGGER AS $$
	//BEGIN
	//  IF OLD.container_ips IS DISTINCT FROM NEW.container_ips THEN
	//	  IF NOT array_compare_as_set(OLD.container_ips, NEW.container_ips) THEN
	//		  UPDATE data_versions set version_id=uuid_generate_v4(), updated_at=NOW() where type_id = 1;
	//	  END IF;
	//  END IF;
	//  IF OLD.svc_fqdn IS DISTINCT FROM NEW.svc_fqdn THEN
	//  	UPDATE data_versions set version_id=uuid_generate_v4(), updated_at=NOW() where type_id = 1;
	//  END IF;
	//  RETURN NEW;
	//END;
	//$$ LANGUAGE plpgsql;
	//
	//CREATE OR REPLACE FUNCTION ports_insert_delete_trigger()
	//RETURNS TRIGGER AS $$
	//BEGIN
	//  IF TG_OP = 'INSERT' THEN
	//	  UPDATE data_versions set version_id=uuid_generate_v4(), updated_at=NOW() where type_id = 1;
	//	  RETURN NEW;
	//
	//  ELSIF TG_OP = 'DELETE' THEN
	//	  UPDATE data_versions set version_id=uuid_generate_v4(), updated_at=NOW() where type_id = 1;
	//	  RETURN OLD;
	//  END IF;
	//
	//  RETURN NULL;
	//END;
	//$$ LANGUAGE plpgsql;
	//
	//DROP TRIGGER IF EXISTS upstreams_update ON upstreams;
	//
	//CREATE TRIGGER upstreams_update
	//    AFTER UPDATE ON upstreams
	//    FOR EACH ROW
	//EXECUTE FUNCTION upstreams_update_trigger();`
)

type DbCfg struct {
	host     string
	port     int
	user     string
	password string
	dbName   string
}

func NewDbCfg(host string, port int, user, pass, dbName string, log *zap.Logger) *DbCfg {
	logger = log
	return &DbCfg{
		host:     host,
		port:     port,
		user:     user,
		password: pass,
		dbName:   dbName,
	}
}

func (c *DbCfg) dsn() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.host, c.port, c.user, c.password, c.dbName)
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
	//dbConn = dbConn.Debug()
	if err := migrate(dbConn); err != nil {
		return nil, err
	}
	if err := seed(dbConn); err != nil {
		return nil, err
	}
	logger.Info("db connection established")
	return dbConn, nil
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&Application{},
		&Protection{},
		&Upstream{},
		&Ingress{},
		&Port{},
		&DataVersion{},
	); err != nil {
		return err
	}
	migrated = true // mark as migrated
	return nil
}

func seed(db *gorm.DB) error {

	if err := db.FirstOrCreate(&DataVersion{TypeId: 1}).Error; err != nil {
		return err
	}
	rawSQL, err := sql.Triggers()
	if err != nil {
		return err
	}
	if err := db.Exec(rawSQL).Error; err != nil {
		return err
	}
	seeded = true // mark as seeded
	return nil
}

func db() *gorm.DB {
	if dbConn == nil {
		logger.Error("database connection not initialized, you must call NewDb(dbCfg) first")
	}
	if !migrated {
		if err := migrate(dbConn); err != nil {
			logger.Error("failed to migrate database", zap.Error(err))
		}
	}
	if !seeded {
		if err := seed(dbConn); err != nil {
			logger.Error("failed to seed database", zap.Error(err))
		}
	}
	return dbConn
}
