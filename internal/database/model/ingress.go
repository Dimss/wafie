package model

import "time"

type Ingress struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Namespace string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Ingress2 struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Namespace string
	CreatedAt time.Time
	UpdatedAt time.Time
}
