package models

import (
	"time"
)

type Protection struct {
	ID            uint           `gorm:"primaryKey"`
	ApplicationID uint           `gorm:"index"`
	Type          ProtectionType `gorm:"index"`

	DesiredState ProtectionState
	ActualState  ProtectionState
	LastUpdated  time.Time
	Reason       string

	WAFConfig *WafProtectionConfig `gorm:"foreignKey:ProtectionID"`
	// Future types: AuthConfig, etc.
}

type ProtectionType string

const (
	ProtectionTypeNone ProtectionType = "NONE"
	ProtectionTypeWAF  ProtectionType = "WAF"
	// ProtectionTypeAuth ProtectionType = "AUTH"
)

type ProtectionState string

const (
	ProtectionUnspecified ProtectionState = "UNSPECIFIED"
	ProtectionUnprotected ProtectionState = "UNPROTECTED"
	ProtectionProtected   ProtectionState = "PROTECTED"
	ProtectionError       ProtectionState = "ERROR"
)
