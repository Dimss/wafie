package models

import (
	"time"

	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EndpointSlice struct {
	ID           uint `gorm:"primary_key"`
	IP           string
	PodName      string
	Namespace    string
	NodeName     string
	PodUID       string
	Ports        string `gorm:"type:text"`
	UpstreamHost string // Foreign key field referencing Ingress.UpstreamHost
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EndpointSliceSvc struct {
	db            *gorm.DB
	logger        *zap.Logger
	EndpointSlice EndpointSlice
}

func NewEndpointSliceModelSvc(tx *gorm.DB, logger *zap.Logger) *EndpointSliceSvc {
	modelSvc := &EndpointSliceSvc{db: tx, logger: logger}

	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}

	return modelSvc
}

func (s *EndpointSlice) ToProto() *v1.EndpointSlice {
	return &v1.EndpointSlice{
		Ip:           s.IP,
		PodName:      s.PodName,
		Namespace:    s.Namespace,
		NodeName:     s.NodeName,
		PodUid:       s.PodUID,
		Ports:        s.Ports,
		UpstreamHost: s.UpstreamHost,
	}
}

func (s *EndpointSliceSvc) NewEndpointSliceFromRequest(req *v1.CreateEndpointSliceRequest) (*EndpointSlice, error) {
	eps := &EndpointSlice{
		IP:           req.EndpointSlice.Ip,
		PodName:      req.EndpointSlice.PodName,
		Namespace:    req.EndpointSlice.Namespace,
		NodeName:     req.EndpointSlice.NodeName,
		PodUID:       req.EndpointSlice.PodUid,
		Ports:        req.EndpointSlice.Ports,
		UpstreamHost: req.EndpointSlice.UpstreamHost,
	}

	if err := s.db.Create(eps).Error; err != nil {
		return nil, err
	}
	return eps, nil
}
