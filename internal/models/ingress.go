package models

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ID                uint `gorm:"primaryKey"`
	Name              string
	Namespace         string
	Host              string `gorm:"uniqueIndex:idx_ing_host"`
	Port              int32
	Path              string
	UpstreamHost      string
	UpstreamPort      int32
	ContainerPort     int32
	ApplicationID     uint `gorm:"not null"`
	Application       Application
	RawIngressSpec    string `gorm:"type:text"`
	IngressType       uint32
	DiscoveryStatus   uint32
	DiscoveryMessage  string `gorm:"type:text"`
	UpstreamRouteType uint32
	ProxyListenerPort uint32 // immutable, note - it is not a part of clause.AssignmentColumns
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (s *IngressModelSvc) NewIngressFromRequest(req *v1.CreateIngressRequest) error {
	ingress := &Ingress{
		Name:              req.Ingress.Name,
		Namespace:         req.Ingress.Namespace,
		Path:              req.Ingress.Path,
		Host:              req.Ingress.Host,
		Port:              req.Ingress.Port,
		UpstreamHost:      req.Ingress.UpstreamHost,
		UpstreamPort:      req.Ingress.UpstreamPort,
		ContainerPort:     req.Ingress.ContainerPort,
		RawIngressSpec:    req.Ingress.RawIngressSpec,
		IngressType:       uint32(req.Ingress.IngressType),
		ApplicationID:     uint(req.Ingress.ApplicationId),
		DiscoveryMessage:  req.Ingress.DiscoveryMessage,
		DiscoveryStatus:   uint32(req.Ingress.DiscoveryStatus),
		UpstreamRouteType: uint32(req.Ingress.UpstreamRouteType),
	}

	if res := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "host"}},
		DoUpdates: clause.AssignmentColumns(
			[]string{
				"name",
				"namespace",
				"path",
				"port",
				"upstream_host",
				"upstream_port",
				"container_port",
				"ingress_type",
				"discovery_message",
				"discovery_status",
				"upstream_route_type",
			},
		),
	}).Create(ingress); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}
	return nil
}

func (i *Ingress) ToProto() *v1.Ingress {
	return &v1.Ingress{
		Name:              i.Name,
		Namespace:         i.Namespace,
		Path:              i.Path,
		Host:              i.Host,
		UpstreamHost:      i.UpstreamHost,
		UpstreamPort:      i.UpstreamPort,
		ContainerPort:     i.ContainerPort,
		IngressType:       v1.IngressType(i.IngressType),
		DiscoveryMessage:  i.DiscoveryMessage,
		DiscoveryStatus:   v1.DiscoveryStatusType(i.DiscoveryStatus),
		ApplicationId:     int32(i.ApplicationID),
		UpstreamRouteType: v1.UpstreamRouteType(i.UpstreamRouteType),
		ProxyListenerPort: int32(i.ProxyListenerPort),
	}
}

func (i *Ingress) createApplicationIfNotExists(tx *gorm.DB) error {
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
	i.ApplicationID = app.ID
	return nil
}

func (i *Ingress) allocateProxyListenerPort(tx *gorm.DB) error {
	// TODO: add test for this stuff!
	allocationAttempts := 10
	for allocationAttempts > 0 {
		proxyListenerPort := getRandomPort()
		ingress := &Ingress{}
		if err := tx.Where("proxy_listener_port = ?", proxyListenerPort).First(ingress).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				i.ProxyListenerPort = proxyListenerPort
				return nil
			}
			return err
		}
		allocationAttempts--
	}
	return fmt.Errorf("error allocating proxy listener port, allocations attempts exceeded")
}

func (i *Ingress) BeforeCreate(tx *gorm.DB) error {
	if err := i.createApplicationIfNotExists(tx); err != nil {
		return err
	}
	if err := i.allocateProxyListenerPort(tx); err != nil {
		return err
	}
	return nil
}

func getRandomPort() uint32 {
	rand.NewSource(time.Now().UnixNano())
	minPort := 49152
	maxPort := 65535
	return uint32(rand.Intn(maxPort-minPort) + minPort)
}
