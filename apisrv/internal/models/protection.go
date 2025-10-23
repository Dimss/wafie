package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProtectionModelSvc struct {
	db         *gorm.DB
	logger     *zap.Logger
	Protection Protection
}

type ModSec struct {
	Mode          uint32 `json:"protectionMode"`
	ParanoiaLevel uint32 `json:"paranoiaLevel"`
}

type ProtectionDesiredState struct {
	ModSec *ModSec `json:"modSec"`
}

type Protection struct {
	ID            uint                   `gorm:"primaryKey"`
	Mode          uint32                 `gorm:"default:0"`
	ApplicationID uint                   `gorm:"not null;uniqueIndex:idx_protection_app_id"`
	Application   Application            `gorm:"foreignKey:ApplicationID"`
	DesiredState  ProtectionDesiredState `gorm:"type:jsonb"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewProtectionModelSvc(tx *gorm.DB, logger *zap.Logger) *ProtectionModelSvc {
	modelSvc := &ProtectionModelSvc{db: tx, logger: logger}

	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}

	return modelSvc
}

func (s *ProtectionDesiredState) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type for ProtectionDesiredState")
	}
}

func (s *ProtectionDesiredState) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *ProtectionDesiredState) FromProto(v1desiredState *wv1.ProtectionDesiredState) {
	s.ModSec = &ModSec{
		Mode:          uint32(v1desiredState.ModeSec.ProtectionMode),
		ParanoiaLevel: uint32(v1desiredState.ModeSec.ParanoiaLevel),
	}
}

func (s *ProtectionDesiredState) ToProto() *wv1.ProtectionDesiredState {
	return nil
}

func (p *Protection) FromProto(protectionv1 *wv1.Protection) error {
	if protectionv1 == nil {
		return fmt.Errorf("protection is required")
	}
	p.Mode = uint32(protectionv1.ProtectionMode)
	p.ApplicationID = uint(protectionv1.ApplicationId)
	p.DesiredState.FromProto(protectionv1.DesiredState)
	return nil
}

func (p *Protection) ToProto() *wv1.Protection {

	protection := &wv1.Protection{
		Id:             uint32(p.ID),
		ApplicationId:  uint32(p.ApplicationID),
		ProtectionMode: wv1.ProtectionMode(p.Mode),
		DesiredState: &wv1.ProtectionDesiredState{ModeSec: &wv1.ModSec{
			ProtectionMode: wv1.ProtectionMode(p.DesiredState.ModSec.Mode),
			ParanoiaLevel:  wv1.ParanoiaLevel(p.DesiredState.ModSec.ParanoiaLevel),
		}},
	}
	//if len(p.Application.Ingress) > 0 {
	//
	//	protection.Upstream = p.Application.Ingress[0].Upstream.ToProto()
	//
	//	protection.Application.Ingress[0] = p.Application.Ingress[0].Upstream.ToProto()
	//}
	if p.Application.ID != 0 {
		protection.Application = p.Application.ToProto()
	}
	return protection
}

func (s *ProtectionModelSvc) CreateProtection(req *wv1.CreateProtectionRequest) (*Protection, error) {
	protection := &Protection{
		ApplicationID: uint(req.ApplicationId),
		Mode:          uint32(req.ProtectionMode),
	}
	protection.DesiredState.FromProto(req.DesiredState)
	if err := s.db.Create(protection).Error; err != nil {
		return nil, err
	}
	return protection, nil
}

func (s *ProtectionModelSvc) GetProtection(req *wv1.GetProtectionRequest) (*Protection, error) {
	protection := &Protection{ID: uint(req.GetId())}
	err := s.db.First(protection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("protection not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return protection, nil
}

func (s *ProtectionModelSvc) UpdateProtection(req *wv1.PutProtectionRequest) (*Protection, error) {
	protection := &Protection{ID: uint(req.GetId())}
	if req.ProtectionMode != nil {
		protection.Mode = uint32(*req.ProtectionMode)
	}
	if req.DesiredState != nil {
		desiredState := &ProtectionDesiredState{}
		desiredState.FromProto(req.DesiredState)
		protection.DesiredState = *desiredState
	}
	// fetch the application id for the given protection
	res := s.db.Model(&Protection{}).
		Select("application_id").
		Where("id = ?", protection.ID).
		Scan(&protection.ApplicationID)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) || res.RowsAffected == 0 {
			return nil, connect.NewError(connect.CodeNotFound, res.Error)
		}
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, res.Error)
	}
	res = s.db.
		Model(protection).
		Updates(protection)
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("protection id not found"))
	}

	return s.GetProtection(&wv1.GetProtectionRequest{Id: uint32(protection.ID)})
}

func (s *ProtectionModelSvc) ListProtections(options *wv1.ListProtectionsOptions) ([]*Protection, error) {
	if options == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("options are required"))
	}
	var protections []*Protection
	query := s.db.Model(&Protection{})
	if options.ProtectionMode != nil {
		query = query.Where("protections.mode = ?", uint32(*options.ProtectionMode))
	}
	if options.ModSecMode != nil {
		query = query.Where(
			fmt.Sprintf(
				"protections.desired_state -> 'modSec' ->> 'protectionMode' = '%d'",
				uint32(*options.ModSecMode),
			),
		)
	}
	if options.IncludeApps != nil && *options.IncludeApps {
		query = query.
			Joins("JOIN applications ON protections.application_id = applications.id").
			Joins("JOIN ingresses ON ingresses.application_id = applications.id").
			Joins("JOIN upstreams ON upstreams.id = ingresses.upstream_id").
			Preload("Application").
			Preload("Application.Ingress").
			Preload("Application.Ingress.Upstream")

		//if options.UpstreamHost != nil {
		//	query = query.Where("ingresses.upstream_host = ?", options.UpstreamHost)
		//}
	}
	res := query.Find(&protections)
	return protections, res.Error
}

func (s *ProtectionModelSvc) DeleteProtection(protectionId uint32) error {
	return s.db.Delete(&Protection{ID: uint(protectionId)}).Error
}

func (p *Protection) AfterCreate(tx *gorm.DB) (err error) {
	err = NewDataVersionModelSvc(tx, nil).UpdateProtectionVersion()
	return err
}

func (p *Protection) AfterUpdate(tx *gorm.DB) (err error) {
	err = NewDataVersionModelSvc(tx, nil).UpdateProtectionVersion()
	return err
}
