package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"math/rand"
	"time"

	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Port struct {
	PortNumber         uint32 `json:"port_number"`
	PortName           string `json:"port_name"`
	Status             uint32 `json:"status"`
	ProxyListeningPort uint32 `json:"proxy_listening_port"` // in use by ContainerPorts only
	Description        string `json:"description"`
}

func (p *Port) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("unsupported type for Port")
	}
}

func (p *Port) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Port) ToProto() *v1.Port {
	return &v1.Port{
		Number:      p.PortNumber,
		Name:        p.PortName,
		Status:      v1.PortStatusType(p.Status),
		Description: p.Description,
	}
}

func NewPortFromProto(port *v1.Port) Port {
	return Port{
		PortNumber:  port.Number,
		PortName:    port.Name,
		Status:      uint32(port.Status),
		Description: port.Description,
	}
}

func NewPortsFromProto(protPorts []*v1.Port) (ports []Port) {
	ports = make([]Port, len(protPorts))
	for idx, port := range protPorts {
		ports[idx] = NewPortFromProto(port)

	}
	return ports
}

type Upstream struct {
	ID                uint           `gorm:"primaryKey"`
	SvcFqdn           string         `gorm:"uniqueIndex:idx_svc_fqdn"`
	ContainerIps      pq.StringArray `gorm:"type:text[]"` // upstream IPS
	SvcPorts          []Port         `gorm:"type:jsonb"`
	ContainerPorts    []Port         `gorm:"type:jsonb"` // upstream ports - for routing by Virtual Host
	UpstreamRouteType uint32
	// One-to-many relationship
	Ingresses []Ingress `gorm:"foreignKey:UpstreamID"` // upstream domains came from Ingres definition
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type UpstreamSvc struct {
	db       *gorm.DB
	logger   *zap.Logger
	Upstream Upstream
}

func NewUpstreamModelSvc(tx *gorm.DB, logger *zap.Logger) *UpstreamSvc {
	modelSvc := &UpstreamSvc{db: tx, logger: logger}
	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}
	return modelSvc
}

func NewUpstreamFromRequest(upstreamReq *v1.Upstream) *Upstream {
	var ingresses = make([]Ingress, len(upstreamReq.Ingresses))
	for idx, ing := range upstreamReq.Ingresses {
		ingresses[idx] = *NewIngressFromProto(ing)
	}
	return &Upstream{
		SvcFqdn:           upstreamReq.SvcFqdn,
		ContainerIps:      upstreamReq.ContainerIps,
		SvcPorts:          NewPortsFromProto(upstreamReq.SvcPorts),
		ContainerPorts:    NewPortsFromProto(upstreamReq.ContainerPorts),
		UpstreamRouteType: uint32(upstreamReq.UpstreamRouteType),
		Ingresses:         ingresses,
	}
}

func (s *UpstreamSvc) Save(u *Upstream) error {
	return s.db.Create(u).Error
}

func (u *Upstream) BeforeCreate(tx *gorm.DB) error {
	if err := u.allocateProxyListenerPort(tx); err != nil {
		return err
	}
	return nil
}

func (u *Upstream) allocateProxyListenerPort(tx *gorm.DB) error {

	// TODO: add test for this stuff!

	existingUpstream := &Upstream{SvcFqdn: u.SvcFqdn}
	if err := tx.First(existingUpstream).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	for idx, port := range u.ContainerPorts {
		if port.ProxyListeningPort != 0 {
			continue
		}
		allocationAttempts := 10
		for allocationAttempts > 0 {
			proxyListenerPort := func() int32 {
				rand.NewSource(time.Now().UnixNano())
				minPort := 49152
				maxPort := 65535
				return int32(rand.Intn(maxPort-minPort) + minPort)
			}()
			query := "container_ports -> 'proxy_listening_port' = ?"
			if err := tx.Where(query, proxyListenerPort).First(&Upstream{}).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					u.ContainerPorts[idx].ProxyListeningPort = uint32(proxyListenerPort)
					break
				}
				return err
			}
			allocationAttempts--
		}
	}
	return nil
}
