package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
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

func NewUpstreamFromRequest(upstreamReq *wv1.Upstream) *Upstream {
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

func NewPortFromProto(port *wv1.Port) Port {
	return Port{
		PortNumber:  port.Number,
		PortName:    port.Name,
		Status:      uint32(port.Status),
		Description: port.Description,
	}
}

func NewPortsFromProto(ports []*wv1.Port) PortSlice {
	portSlice := make(PortSlice, len(ports))
	for idx, port := range ports {
		portSlice[idx] = NewPortFromProto(port)
	}
	return portSlice
}

func (p *Port) ToProto() *wv1.Port {
	return &wv1.Port{
		Number:      p.PortNumber,
		Name:        p.PortName,
		Status:      wv1.PortStatusType(p.Status),
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

//goland:noinspection GoMixedReceiverTypes
func (p *PortSlice) ToProto() []*wv1.Port {
	ports := make([]*wv1.Port, len(*p))
	for idx, port := range *p {
		ports[idx] = port.ToProto()
	}
	return ports
}

func (s *UpstreamSvc) Save(u *Upstream, options *wv1.CreateUpstreamOptions) error {
	// by default do not upsert container_ips
	assigmentColumns := []string{
		"svc_ports",
		"container_ports",
		"upstream_route_type",
		"created_at",
		"updated_at",
	}
	// if set container ips only is true update only container_ips column
	if options != nil && *options.SetContainerIpsOnly {
		assigmentColumns = []string{
			"container_ips",
			"created_at",
			"updated_at",
		}
	}
	if res := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "svc_fqdn"}},
		DoUpdates: clause.AssignmentColumns(assigmentColumns),
	}).Omit("Ingresses").Create(&u); res.Error != nil {
		return connect.NewError(connect.CodeUnknown, res.Error)
	}
	return nil
}

func (s *UpstreamSvc) List(options *wv1.ListUpstreamsOptions) (upstreams []*Upstream, err error) {
	query := s.db.Model(&Upstream{})
	if options != nil && options.IncludeIngress != nil && *options.IncludeIngress {
		query = query.
			Joins("JOIN ingresses ON ingresses.upstream_id = upstreams.id").
			Preload("Ingresses")
	}

	return upstreams, query.Find(&upstreams).Error
}

func (u *Upstream) ToProto() *wv1.Upstream {
	wv1upstream := &wv1.Upstream{
		SvcFqdn:           u.SvcFqdn,
		ContainerIps:      u.ContainerIps,
		SvcPorts:          u.SvcPorts.ToProto(),
		ContainerPorts:    u.ContainerPorts.ToProto(),
		Ingresses:         nil,
		UpstreamRouteType: wv1.UpstreamRouteType(u.UpstreamRouteType),
	}
	if u.Ingresses != nil {
		wv1upstream.Ingresses = make([]*wv1.Ingress, len(u.Ingresses))
		for idx, ingress := range u.Ingresses {
			wv1upstream.Ingresses[idx] = ingress.ToProto()
		}
	}
	return wv1upstream
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
	// set default upstream route type
	if u.UpstreamRouteType == uint32(wv1.UpstreamRouteType_UPSTREAM_ROUTE_TYPE_UNSPECIFIED) {
		u.UpstreamRouteType = uint32(wv1.UpstreamRouteType_UPSTREAM_ROUTE_TYPE_PORT)
	}
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
	if existingUpstream.ContainerPorts != nil {
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
			query := `SELECT (port ->> 'proxy_listening_port')::int as proxy_listening_port 
                        FROM upstreams,jsonb_array_elements(container_ports) as port 
                        where (port ->> 'proxy_listening_port')::int = ?`
			var ports []string
			if err := tx.Raw(query, proxyListenerPort).Pluck("jsonb_array_elements", &ports).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					u.ContainerPorts[idx].ProxyListeningPort = uint32(proxyListenerPort)
					break
				}
				return err
			}
			if len(ports) == 0 {
				u.ContainerPorts[idx].ProxyListeningPort = uint32(proxyListenerPort)
				break
			}
			allocationAttempts--
		}
	}
	return nil
}
