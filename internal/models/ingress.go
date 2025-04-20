package models

import (
	"connectrpc.com/connect"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"time"
)

type Ingress struct {
	ID            uint `gorm:"primaryKey"`
	Name          string
	Host          string `gorm:"uniqueIndex:idx_ing_host"`
	Port          string
	Path          string
	UpstreamHost  string
	UpstreamPort  int32
	ApplicationID uint `gorm:"not null"`
	Application   Application
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewIngressFromRequest(req *v1.CreateIngressRequest, app *Application) error {
	ingress := &Ingress{
		Name:          req.Ingress.Name,
		Path:          req.Ingress.Path,
		Host:          req.Ingress.Host,
		UpstreamHost:  req.Ingress.UpstreamHost,
		UpstreamPort:  req.Ingress.UpstreamPort,
		ApplicationID: app.ID,
	}
	if res := db().Create(ingress); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}
	return nil
}

func (i *Ingress) ToProto() *v1.Ingress {
	return &v1.Ingress{
		Name:         i.Name,
		Path:         i.Path,
		Host:         i.Host,
		UpstreamHost: i.UpstreamHost,
		UpstreamPort: i.UpstreamPort,
	}
}
