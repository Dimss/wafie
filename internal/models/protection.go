package models

import (
	"connectrpc.com/connect"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
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
	ID               uint                   `gorm:"primaryKey"`
	Mode             uint32                 `gorm:"default:0"`
	IngressAutoPatch uint32                 `gorm:"default:2"`
	ApplicationID    uint                   `gorm:"not null;uniqueIndex:idx_protection_app_id"`
	Application      Application            `gorm:"foreignKey:ApplicationID"`
	DesiredState     ProtectionDesiredState `gorm:"type:jsonb"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
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

func (s *ProtectionDesiredState) FromProto(v1desiredState *v1.ProtectionDesiredState) {
	s.ModSec = &ModSec{
		Mode:          uint32(v1desiredState.ModeSec.ProtectionMode),
		ParanoiaLevel: uint32(v1desiredState.ModeSec.ParanoiaLevel),
	}
}

func (s *ProtectionDesiredState) ToProto() *v1.ProtectionDesiredState {
	return nil
}

func (p *Protection) FromProto(protectionv1 *v1.Protection) error {
	if protectionv1 == nil {
		return fmt.Errorf("protection is required")
	}
	p.Mode = uint32(protectionv1.ProtectionMode)
	p.IngressAutoPatch = uint32(protectionv1.IngressAutoPatch)
	p.ApplicationID = uint(protectionv1.ApplicationId)
	p.DesiredState.FromProto(protectionv1.DesiredState)
	return nil
}

func (p *Protection) ToProto() *v1.Protection {

	protection := &v1.Protection{
		Id:               uint32(p.ID),
		ApplicationId:    uint32(p.ApplicationID),
		ProtectionMode:   v1.ProtectionMode(p.Mode),
		IngressAutoPatch: v1.IngressAutoPatch(p.IngressAutoPatch),
		DesiredState: &v1.ProtectionDesiredState{ModeSec: &v1.ModSec{
			ProtectionMode: v1.ProtectionMode(p.DesiredState.ModSec.Mode),
			ParanoiaLevel:  v1.ParanoiaLevel(p.DesiredState.ModSec.ParanoiaLevel),
		}},
	}
	if p.Application.ID != 0 {
		protection.Application = p.Application.ToProto()
	}
	return protection
}

func (s *ProtectionModelSvc) CreateProtection(req *v1.CreateProtectionRequest) (*Protection, error) {
	protection := &Protection{
		ApplicationID:    uint(req.ApplicationId),
		Mode:             uint32(req.ProtectionMode),
		IngressAutoPatch: uint32(req.IngressAutoPatch),
	}
	protection.DesiredState.FromProto(req.DesiredState)
	if err := s.db.Create(protection).Error; err != nil {
		return nil, err
	}
	return protection, nil
}

func (s *ProtectionModelSvc) GetProtection(req *v1.GetProtectionRequest) (*Protection, error) {
	protection := &Protection{ID: uint(req.GetId())}
	err := s.db.First(protection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("protection not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return protection, nil
}

func (s *ProtectionModelSvc) UpdateProtection(req *v1.PutProtectionRequest) (*Protection, error) {
	protection := &Protection{ID: uint(req.GetId())}
	if req.ProtectionMode != nil {
		protection.Mode = uint32(*req.ProtectionMode)
	}
	if req.DesiredState != nil {
		desiredState := &ProtectionDesiredState{}
		desiredState.FromProto(req.DesiredState)
		protection.DesiredState = *desiredState
	}
	if req.IngressAutoPatch != nil {
		protection.IngressAutoPatch = uint32(*req.IngressAutoPatch)
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

	return s.GetProtection(&v1.GetProtectionRequest{Id: uint32(protection.ID)})
}

func (s *ProtectionModelSvc) ListProtections(options *v1.ListProtectionsOptions) ([]*Protection, error) {
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
			Preload("Application").
			Preload("Application.Ingress")
	}
	res := query.Find(&protections)
	return protections, res.Error
}

func (p *Protection) AfterCreate(tx *gorm.DB) (err error) {
	//vhModelSvc := NewVirtualHostModelSvc(tx, nil)
	//_, err := vhModelSvc.CreateVirtualHost(p.ID)
	err = NewDataVersionModelSvc(tx, nil).UpdateProtectionVersion()
	return err
}

func (p *Protection) AfterUpdate(tx *gorm.DB) (err error) {
	//vhModelSvc := NewVirtualHostModelSvc(tx, nil)
	err = NewDataVersionModelSvc(tx, nil).UpdateProtectionVersion()
	//virtualHost, err := vhModelSvc.GetVirtualHostByProtectionId(p.ID)
	//if connect.CodeOf(err) == connect.CodeNotFound {
	//	if _, err := vhModelSvc.CreateVirtualHost(p.ID); err != nil {
	//		return err
	//	}
	//	// update protection data version
	//	if err := dataVersionSvc.UpdateProtectionVersion(); err != nil {
	//		return err
	//	}
	//	return nil
	//} else if err != nil {
	//	return err // unexpected error, rollback transaction and return an error
	//}
	//if _, err := vhModelSvc.UpdateVirtualHost(virtualHost.ID); err != nil {
	//	return err
	//}
	//if err := dataVersionSvc.UpdateProtectionVersion(); err != nil {
	//	return err
	//}
	return err
}
