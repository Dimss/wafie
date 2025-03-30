package models

type Service struct {
	ID uint `gorm:"primaryKey"`
}

type K8sServiceVersion struct {
	ID        uint `gorm:"primaryKey"`
	Spec      string
	Version   uint
	IngressID uint
	Ingress   Ingress
}
