package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"go.uber.org/zap"
)

type ModSecProtection struct {
	Enabled       bool   `json:"enabled"`
	ParanoiaLevel string `json:"paranoiaLevel"`
}

type ProtectionDesiredState struct {
	ModSecProtection *ModSecProtection `json:"modSecProtection"`
}

type Protection struct {
	ID            uint                   `gorm:"primaryKey"`
	Enabled       bool                   `gorm:"default:false"`
	ApplicationID uint                   `gorm:"not null;uniqueIndex:idx_protection_app_id"`
	Application   Application            `gorm:"foreignKey:ApplicationID"`
	DesiredState  ProtectionDesiredState `gorm:"type:jsonb"`
}

// Scan implements the sql.Scanner interface for reading JSONB from the database
func (p *ProtectionDesiredState) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("unsupported type for ProtectionDesiredState")
	}
}

// Value implements the driver.Valuer interface for writing JSONB to the database
func (p *ProtectionDesiredState) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func CreateProtection(req *v1.CreateProtectionRequest) (*Protection, error) {
	protection := &Protection{}
	if err := protection.FromProto(req); err != nil {
		return nil, err
	}
	if err := db().Create(protection).Error; err != nil {
		return nil, err
	}
	return protection, nil
}

func (p *Protection) FromProto(req *v1.CreateProtectionRequest) error {
	if req.Protection == nil {
		return fmt.Errorf("protection is required")
	}
	p.Enabled = req.Protection.Enabled
	p.ApplicationID = uint(req.Protection.ApplicationId)
	if req.Protection.DesiredState != nil {
		p.DesiredState = ProtectionDesiredState{
			ModSecProtection: &ModSecProtection{
				ParanoiaLevel: req.Protection.DesiredState.ModeSecProtection.ParanoiaLevel,
			},
		}
		if req.Protection.DesiredState.ModeSecProtection.ProtectionMode ==
			v1.ModSecProtectionMode_MOD_SEC_PROTECTION_MODE_ON {
			p.DesiredState.ModSecProtection.Enabled = true
		}
	}
	return nil
}

func CreateAnProtection() {
	// Create a new protection
	protection := Protection{
		ApplicationID: 1,
		DesiredState: ProtectionDesiredState{
			ModSecProtection: &ModSecProtection{
				Enabled:       true,
				ParanoiaLevel: "p4",
			},
		},
	}

	mlog().Info("Creating protection", zap.Any("protection", protection))

	// Save the protection to the database
	if err := db().Create(&protection).Error; err != nil {
		fmt.Println("Error creating protection:", err)
	} else {
		fmt.Println("Protection created successfully")
	}
	p := &Protection{}
	res := db().First(p, 1)
	if res.Error != nil {
		fmt.Println("Error fetching protection:", res.Error)
	}
	fmt.Println("Fetched protection:", p)
}
