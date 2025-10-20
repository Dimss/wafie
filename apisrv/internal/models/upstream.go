package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Port struct {
	ID                 uint   `gorm:"primaryKey"`
	PortNumber         uint32 `json:"port_number"`
	PortName           string `json:"port_name"`
	Status             uint32 `json:"status"`
	ProxyListeningPort uint32 `json:"proxy_listening_port"` // in use by ContainerPorts only
	Description        string `json:"description"`
}

type PortSlice []Port

type Upstream struct {
	ID                uint           `gorm:"primaryKey"`
	SvcFqdn           string         `gorm:"uniqueIndex:idx_svc_fqdn"`
	ContainerIps      pq.StringArray `gorm:"type:text[]"` // upstream IPS
	SvcPorts          PortSlice      `gorm:"type:jsonb"`
	ContainerPorts    PortSlice      `gorm:"type:jsonb"` // upstream ports - for routing by Virtual Host
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

func NewPortFromProto(port *v1.Port) Port {
	return Port{
		PortNumber:  port.Number,
		PortName:    port.Name,
		Status:      uint32(port.Status),
		Description: port.Description,
	}
}

func NewPortsFromProto(protPorts []*v1.Port) (ports PortSlice) {
	ports = make([]Port, len(protPorts))
	for idx, port := range protPorts {
		ports[idx] = NewPortFromProto(port)

	}
	return ports
}

func (p *Port) ToProto() *v1.Port {
	return &v1.Port{
		Number:      p.PortNumber,
		Name:        p.PortName,
		Status:      v1.PortStatusType(p.Status),
		Description: p.Description,
	}
}

//goland:noinspection GoMixedReceiverTypes
func (p PortSlice) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	jsonData, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PortSlice to JSON: %w", err)
	}
	return string(jsonData), nil
}

//goland:noinspection GoMixedReceiverTypes
func (p *PortSlice) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}
	var jsonData []byte
	switch v := value.(type) {
	case []byte:
		jsonData = v
	case string:
		jsonData = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into PortSlice", value)
	}
	// Unmarshal JSON to PortSlice
	if err := json.Unmarshal(jsonData, p); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to PortSlice: %w", err)
	}

	return nil
}

func (s *UpstreamSvc) Save(u *Upstream) error {
	if res := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "svc_fqdn"}},
		DoUpdates: clause.AssignmentColumns(
			[]string{
				"container_ips",
				"svc_ports",
				"container_ports",
				"upstream_route_type",
				"created_at",
				"updated_at",
			},
		),
	}).Omit("Ingresses").Create(&u); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}
	return nil
}

func (u *Upstream) AfterSave(tx *gorm.DB) error {
	if u.Ingresses != nil {
		// TODO: improve to batch operation
		ingressModelSvc := NewIngressModelSvc(tx, nil)
		for _, ing := range u.Ingresses {
			ing.UpstreamID = u.ID
			if err := ingressModelSvc.Save(&ing); err != nil {
				return err
			}
		}
	}
	return nil
}

func (u *Upstream) BeforeCreate(tx *gorm.DB) error {
	if err := u.allocateProxyListenerPort(tx); err != nil {
		return err
	}
	return nil
}

func (u *Upstream) updateCurrentProxyListeningPorts(tx *gorm.DB) error {
	existingUpstream := &Upstream{}
	if err := tx.Where("svc_fqdn = ?", u.SvcFqdn).First(existingUpstream).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	for _, currentPort := range existingUpstream.ContainerPorts {
		if currentPort.ProxyListeningPort == 0 {
			continue
		}
		for idx, newPort := range u.ContainerPorts {
			if (currentPort.PortNumber != 0 && currentPort.PortNumber == newPort.PortNumber) ||
				(currentPort.PortName != "" && currentPort.PortName == newPort.PortName) {
				u.ContainerPorts[idx].ProxyListeningPort = currentPort.ProxyListeningPort
				break
			}
		}
	}
	return nil
}

// TODO: TEST THIS WITH UNIT TESTS!
func (u *Upstream) allocateProxyListenerPort(tx *gorm.DB) error {
	if err := u.updateCurrentProxyListeningPorts(tx); err != nil {
		return err
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
