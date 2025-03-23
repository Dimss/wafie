package model

import (
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"time"
)

type Ingress struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex:idx_name_namespace"`
	Namespace   string `gorm:"uniqueIndex:idx_name_namespace"`
	PortNumber  int32
	PortName    string
	Path        string
	Host        string
	ServiceName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewIngressFromRequest(req *v1.CreateIngressRequest) *Ingress {
	return &Ingress{
		Name:        req.GetName(),
		Namespace:   req.GetNamespace(),
		PortNumber:  req.GetPortNumber(),
		PortName:    req.GetPortName(),
		Path:        req.GetPath(),
		Host:        req.GetHost(),
		ServiceName: req.GetServiceName(),
	}
}
