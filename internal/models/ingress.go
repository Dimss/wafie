package models

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
	ApplicationID uint `gorm:"not null"`
	Application   Application
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewIngressFromRequest(req *v1.CreateIngressRequest, app *Application) error {
	ingress := &Ingress{
		Name:          req.Ingress.GetName(),
		Namespace:     req.Ingress.GetNamespace(),
		PortNumber:    req.Ingress.GetPortNumber(),
		PortName:      req.Ingress.GetPortName(),
		Path:          req.Ingress.GetPath(),
		Host:          req.Ingress.GetHost(),
		ServiceName:   req.Ingress.GetServiceName(),
		ApplicationID: app.ID,
	}
	if res := db().Create(ingress); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}

	return nil
}
func (i *Ingress) ToProto() *v1.Ingress {
	return &v1.Ingress{
		Name:        i.Name,
		Namespace:   i.Namespace,
		PortNumber:  i.PortNumber,
		PortName:    i.PortName,
		Path:        i.Path,
		Host:        i.Host,
		ServiceName: i.ServiceName,
	}
}
