package models

import (
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DataVersionModelSvc struct {
	db          *gorm.DB
	logger      *zap.Logger
	DataVersion DataVersion
}

type DataVersion struct {
	TypeId    uint32 `gorm:"primaryKey"`
	VersionId string
}

func NewDataVersionModelSvc(tx *gorm.DB, logger *zap.Logger) *DataVersionModelSvc {
	modelSvc := &DataVersionModelSvc{db: tx, logger: logger}
	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}
	return modelSvc
}

func (s *DataVersionModelSvc) GetVersionByTypeId(typeId uint32) (*DataVersion, error) {
	dv := &DataVersion{}
	if err := s.db.First(dv, typeId).Error; err != nil {
		return nil, err
	}
	return dv, nil
}

func (s *DataVersionModelSvc) UpdateProtectionVersion() error {
	return s.db.Save(
		&DataVersion{
			TypeId:    uint32(v1.DataTypeId_DATA_TYPE_ID_PROTECTION),
			VersionId: uuid.New().String(),
		},
	).Error
}

func (d *DataVersion) ToProto() *v1.GetDataVersionResponse {
	return &v1.GetDataVersionResponse{
		TypeId:    v1.DataTypeId(d.TypeId),
		VersionId: d.VersionId,
	}
}
