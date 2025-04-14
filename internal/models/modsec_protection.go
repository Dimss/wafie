package models

type ModSecProtectionConfig struct {
	ID            uint   `gorm:"primaryKey"`
	Enabled       string `gorm:"default:on"`
	ApplicationID uint
	Application   Application
}
