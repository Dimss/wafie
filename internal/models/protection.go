package models

import (
	"connectrpc.com/connect"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"gorm.io/gorm"
)

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

func CreateProtection(req *v1.CreateProtectionRequest) (*Protection, error) {
	protection := &Protection{}
	if err := protection.FromProto(req.Protection); err != nil {
		return nil, err
	}
	if err := db().Create(protection).Error; err != nil {
		return nil, err
	}
	return protection, nil
}

func GetProtection(req *v1.GetProtectionRequest) (*Protection, error) {
	protection := &Protection{ID: uint(req.GetId())}
	err := db().First(protection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("protection not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return protection, nil
}

func UpdateProtection(req *v1.PutProtectionRequest) (*Protection, error) {
	desiredState := &ProtectionDesiredState{}
	desiredState.FromProto(req.DesiredState)
	protection := &Protection{
		ID:           uint(req.GetId()),
		Mode:         uint32(req.ProtectionMode),
		DesiredState: *desiredState,
	}

	// fetch the application id for the given protection
	if res := db().Model(&Protection{}).
		Select("application_id").
		Where("id = ?", protection.ID).
		Scan(&protection.ApplicationID); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, res.Error)
		}
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}

	res := db().
		Model(protection).
		Where("id = ?", protection.ID).
		Updates(protection)
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("protection id not found"))
	}
	return protection, nil
}

func (p *Protection) FromProto(protectionv1 *v1.Protection) error {
	if protectionv1 == nil {
		return fmt.Errorf("protection is required")
	}
	p.Mode = uint32(protectionv1.ProtectionMode)
	p.ApplicationID = uint(protectionv1.ApplicationId)
	if protectionv1.DesiredState != nil {
		p.DesiredState = ProtectionDesiredState{
			ModSec: &ModSec{
				ParanoiaLevel: uint32(protectionv1.DesiredState.ModeSec.ParanoiaLevel),
				Mode:          uint32(protectionv1.DesiredState.ModeSec.ProtectionMode),
			},
		}

	}
	return nil
}

func (p *Protection) ToProto() *v1.Protection {

	protection := &v1.Protection{
		Id:             uint32(p.ID),
		ApplicationId:  uint32(p.ApplicationID),
		ProtectionMode: v1.ProtectionMode(p.Mode),
		DesiredState: &v1.ProtectionDesiredState{ModeSec: &v1.ModSec{
			ProtectionMode: v1.ProtectionMode(p.DesiredState.ModSec.Mode),
			ParanoiaLevel:  v1.ParanoiaLevel(p.DesiredState.ModSec.ParanoiaLevel),
		}},
	}
	return protection

}

//
//func CreateAnProtection() {
//	// Create a new protection
//	protection := Protection{
//		ApplicationID: 1,
//		DesiredState: ProtectionDesiredState{
//			ModSec: &ModSec{
//				Mode:       true,
//				ParanoiaLevel: "p4",
//			},
//		},
//	}
//
//	mlog().Info("Creating protection", zap.Any("protection", protection))
//
//	// Save the protection to the database
//	if err := db().Create(&protection).Error; err != nil {
//		fmt.Println("Error creating protection:", err)
//	} else {
//		fmt.Println("Protection created successfully")
//	}
//	p := &Protection{}
//	res := db().First(p, 1)
//	if res.Error != nil {
//		fmt.Println("Error fetching protection:", res.Error)
//	}
//	fmt.Println("Fetched protection:", p)
//}
