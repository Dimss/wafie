package models

//
//type ProtectionState string
//type ProtectionType string
//
//const (
//	ProtectionUnspecified ProtectionState = "UNSPECIFIED"
//	ProtectionUnprotected ProtectionState = "UNPROTECTED"
//	ProtectionProtected   ProtectionState = "PROTECTED"
//	ProtectionError       ProtectionState = "ERROR"
//)
//
//const (
//	ProtectionTypeNone ProtectionType = "NONE"
//	ProtectionTypeWAF  ProtectionType = "MODSEC"
//)
//
//type Protection struct {
//	ID            uint
//	ApplicationID uint
//
//	DesiredState ProtectionState
//	ActualState  ProtectionState
//
//	Type   ProtectionType          // Still useful for querying
//	ModSec *ModSecProtectionConfig `gorm:"foreignKey:ProtectionID"`
//}
