package models

import (
	"time"
)

type Protection struct {
	ID            uint
	ApplicationID uint

	DesiredState ProtectionState
	ActualState  ProtectionState
	LastUpdated  time.Time
	Reason       string

	Type      ProtectionType       // Still useful for querying
	WAFConfig *WafProtectionConfig `gorm:"foreignKey:ProtectionID"`
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
