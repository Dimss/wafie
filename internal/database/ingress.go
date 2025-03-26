package database

import (
	"connectrpc.com/connect"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"time"
)

type Ingress struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"uniqueIndex:idx_ing_name_namespace"`
	Namespace     string `gorm:"uniqueIndex:idx_ing_name_namespace"`
	PortNumber    int32
	PortName      string
	Path          string
	Host          string
	ServiceName   string
	ApplicationID uint
	Application   Application
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewIngressFromRequest(req *v1.CreateIngressRequest, app *Application) error {
	ingress := &Ingress{
		Name:          req.GetName(),
		Namespace:     req.GetNamespace(),
		PortNumber:    req.GetPortNumber(),
		PortName:      req.GetPortName(),
		Path:          req.GetPath(),
		Host:          req.GetHost(),
		ServiceName:   req.GetServiceName(),
		ApplicationID: app.ID,
	}
	if res := db().Create(ingress); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}

	return nil
}
