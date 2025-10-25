package models

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Upstream struct {
	ID                uint           `gorm:"primaryKey"`
	SvcFqdn           string         `gorm:"uniqueIndex:idx_svc_fqdn"`
	ContainerIps      pq.StringArray `gorm:"type:text[]"` // upstream IPS
	UpstreamRouteType uint32
	// One-to-many relationship
	Ingresses []Ingress `gorm:"foreignKey:UpstreamID"`
	Ports     []Port    `gorm:"foreignKey:UpstreamID"`
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
		UpstreamRouteType: uint32(upstreamReq.UpstreamRouteType),
		Ingresses:         ingresses,
	}
}

func (s *UpstreamSvc) Save(u *Upstream) (*Upstream, error) {

	assigmentColumns := []string{
		"upstream_route_type",
		"created_at",
		"updated_at",
	}
	if res := s.db.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "svc_fqdn"}},
			DoUpdates: clause.AssignmentColumns(assigmentColumns),
		},
	).
		Omit("Ingresses").
		Omit("Ports").
		Create(&u); res.Error != nil {
		return u, connect.NewError(connect.CodeUnknown, res.Error)
	}
	return u, nil
}

func (s *UpstreamSvc) List(options *wv1.ListRoutesOptions) (upstreams []*Upstream, err error) {
	query := s.db.Model(&Upstream{})
	if options != nil && options.IncludeIngress != nil && *options.IncludeIngress {
		query = query.
			Joins("JOIN ingresses ON ingresses.upstream_id = upstreams.id").
			Joins("JOIN ports ON ports.upstream_id = upstreams.id").
			Preload("Ingresses").
			Preload("Ports")
	}
	if options != nil && options.SvcFqdn != nil {
		query = query.Where("svc_fqdn = ?", options.SvcFqdn)
	}
	return upstreams, query.Distinct().Find(&upstreams).Error

}

func (u *Upstream) ToProto() *wv1.Upstream {
	wv1upstream := &wv1.Upstream{
		SvcFqdn:           u.SvcFqdn,
		ContainerIps:      u.ContainerIps,
		UpstreamRouteType: wv1.UpstreamRouteType(u.UpstreamRouteType),
	}
	if u.Ingresses != nil {
		wv1upstream.Ingresses = make([]*wv1.Ingress, len(u.Ingresses))
		for idx, ingress := range u.Ingresses {
			wv1upstream.Ingresses[idx] = ingress.ToProto()
		}
	}
	if u.Ports != nil {
		wv1upstream.Ports = make([]*wv1.Port, len(u.Ports))
		for idx, port := range u.Ports {
			wv1upstream.Ports[idx] = port.ToProto()
		}
	}
	return wv1upstream
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
	//if existingUpstream.ContainerPorts != nil {
	//	for _, currentPort := range existingUpstream.ContainerPorts {
	//		if currentPort.ProxyListeningPort == 0 {
	//			continue
	//		}
	//		for idx, newPort := range u.ContainerPorts {
	//			if (currentPort.PortNumber != 0 && currentPort.PortNumber == newPort.PortNumber) ||
	//				(currentPort.PortName != "" && currentPort.PortName == newPort.PortName) {
	//				u.ContainerPorts[idx].ProxyListeningPort = currentPort.ProxyListeningPort
	//				break
	//			}
	//		}
	//	}
	//}
	return nil
}

// TODO: TEST THIS WITH UNIT TESTS!
func (u *Upstream) allocateProxyListenerPort(tx *gorm.DB) error {
	//if err := u.updateCurrentProxyListeningPorts(tx); err != nil {
	//	return err
	//}
	//for idx, port := range u.ContainerPorts {
	//	if port.ProxyListeningPort != 0 {
	//		continue
	//	}
	//	allocationAttempts := 10
	//	for allocationAttempts > 0 {
	//		proxyListenerPort := func() int32 {
	//			rand.NewSource(time.Now().UnixNano())
	//			minPort := 49152
	//			maxPort := 65535
	//			return int32(rand.Intn(maxPort-minPort) + minPort)
	//		}()
	//		query := `SELECT (port ->> 'proxy_listening_port')::int as proxy_listening_port
	//                    FROM upstreams,jsonb_array_elements(container_ports) as port
	//                    where (port ->> 'proxy_listening_port')::int = ?`
	//		var ports []string
	//		if err := tx.Raw(query, proxyListenerPort).Pluck("jsonb_array_elements", &ports).Error; err != nil {
	//			return err
	//		}
	//		if len(ports) == 0 {
	//			u.ContainerPorts[idx].ProxyListeningPort = uint32(proxyListenerPort)
	//			if err := NewDataVersionModelSvc(tx, nil).UpdateProtectionVersion(); err != nil {
	//				return err
	//			}
	//			break
	//		}
	//		allocationAttempts--
	//	}
	//
	//}
	return nil
}
