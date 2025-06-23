package models

import (
	"github.com/Dimss/cwaf/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SystemModelSvc struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSystemModelSvc(tx *gorm.DB, logger *zap.Logger) *SystemModelSvc {
	if tx == nil {
		tx = db()
	}
	if logger == nil {
		logger = applogger.NewLogger()
	}
	return &SystemModelSvc{
		db:     tx,
		logger: logger,
	}
}

func (s *SystemModelSvc) Ping() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Ping(); err != nil {
		return err
	}
	return nil
}
