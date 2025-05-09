package models

import (
	"connectrpc.com/connect"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type IngressModelSvc struct {
	db      *gorm.DB
	logger  *zap.Logger
	Ingress Ingress
}

func NewIngressModelSvc(tx *gorm.DB, logger *zap.Logger) *IngressModelSvc {
	modelSvc := &IngressModelSvc{db: tx, logger: logger}

	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}

	return modelSvc
}

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

func (s *IngressModelSvc) NewIngressFromRequest(req *v1.CreateIngressRequest) error {
	ingress := &Ingress{
		Name:          req.Ingress.Name,
		Path:          req.Ingress.Path,
		Host:          req.Ingress.Host,
		UpstreamHost:  req.Ingress.UpstreamHost,
		UpstreamPort:  req.Ingress.UpstreamPort,
		ApplicationID: uint(req.Ingress.ApplicationId),
	}
	if res := s.db.Create(ingress); res.Error != nil {
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

func (i *Ingress) BeforeCreate(tx *gorm.DB) error {
	if i.ApplicationID != 0 {
		return nil
	}
	appModelSvc := NewApplicationModelSvc(tx, nil)
	newAppReq := &v1.CreateApplicationRequest{Name: i.Name}
	appId, err := appModelSvc.CreateApplication(newAppReq)
	if err != nil {
		return err
	}
	i.ApplicationID = appId.ID
	return nil
}
