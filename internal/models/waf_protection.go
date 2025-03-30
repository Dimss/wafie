package models

type WafProtectionConfig struct {
	ID           uint `gorm:"primaryKey"`
	ProtectionID uint `gorm:"uniqueIndex"` // One WAF config per protection

	RuleSet      string
	AllowListIPs GormStringArray `gorm:"type:text"` // JSON-encoded array
}
