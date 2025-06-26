package models

import (
	"connectrpc.com/connect"
	"errors"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ID             uint `gorm:"primaryKey"`
	Name           string
	Namespace      string
	Host           string `gorm:"uniqueIndex:idx_ing_host"`
	Port           string
	Path           string
	UpstreamHost   string
	UpstreamPort   int32
	ApplicationID  uint `gorm:"not null"`
	Application    Application
	RawIngressSpec string `gorm:"type:text"`
	IngressType    uint32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *IngressModelSvc) NewIngressFromRequest(req *v1.CreateIngressRequest) error {
	ingress := &Ingress{
		Name:           req.Ingress.Name,
		Namespace:      req.Ingress.Namespace,
		Path:           req.Ingress.Path,
		Host:           req.Ingress.Host,
		UpstreamHost:   req.Ingress.UpstreamHost,
		UpstreamPort:   req.Ingress.UpstreamPort,
		RawIngressSpec: req.Ingress.RawIngressSpec,
		IngressType:    uint32(req.Ingress.IngressType),
		ApplicationID:  uint(req.Ingress.ApplicationId),
	}
	if res := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "host"}},
		DoUpdates: clause.AssignmentColumns(
			[]string{
				"name",
				"namespace",
				"host",
				"port",
				"path",
				"upstream_host",
				"upstream_port",
			},
		),
	}).Create(ingress); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}
	return nil
}

func (i *Ingress) ToProto() *v1.Ingress {
	return &v1.Ingress{
		Name:           i.Name,
		Namespace:      i.Namespace,
		Path:           i.Path,
		Host:           i.Host,
		UpstreamHost:   i.UpstreamHost,
		UpstreamPort:   i.UpstreamPort,
		RawIngressSpec: i.RawIngressSpec,
		IngressType:    v1.IngressType(i.IngressType),
		ApplicationId:  int32(i.ApplicationID),
	}
}

func (i *Ingress) BeforeCreate(tx *gorm.DB) error {
	app := &Application{}
	if err := tx.Where("name = ?", i.Host).First(app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			appModelSvc := NewApplicationModelSvc(tx, nil)
			newAppReq := &v1.CreateApplicationRequest{Name: i.Host}
			appId, err := appModelSvc.CreateApplication(newAppReq)
			if err != nil {
				return err
			}
			i.ApplicationID = appId.ID
			return nil
		}
		if err != nil {
			return err
		}
	}
	// if application ID is set,
	// meaning this is an explicit update operator
	// nothing needs to be done
	//if i.ApplicationID != 0 {
	//	return nil
	//}
	// on upsert of ingress the application ID might not bet set
	// thus extra checks needs to be done
	//existingIngress := &Ingress{}
	//if err := tx.Where("host = ?", i.Host).First(existingIngress).Error; err != nil {
	//	// if the ingress with the host name not found,
	//	// meaning it is a new ingress request
	//	// new application needs to be created
	//	if errors.Is(err, gorm.ErrRecordNotFound) {
	//		appModelSvc := NewApplicationModelSvc(tx, nil)
	//		newAppReq := &v1.CreateApplicationRequest{Name: i.Host}
	//		appId, err := appModelSvc.CreateApplication(newAppReq)
	//		if err != nil {
	//			return err
	//		}
	//		i.ApplicationID = appId.ID
	//		return nil
	//	} else {
	//		// in case of error, return and do nothing
	//		return err
	//	}
	//}
	// if the ingress with the host name found,
	// meaning application already exists
	// set the application ID and return
	//i.ApplicationID = app.ID
	return nil

}
