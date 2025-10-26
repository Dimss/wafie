package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Endpoint struct {
	NodeName  string `json:"nodeName"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func NewEndpointFromRequest(ep *wv1.Endpoint) Endpoint {
	return Endpoint{
		NodeName:  ep.NodeName,
		Kind:      ep.Kind,
		Name:      ep.Name,
		Namespace: ep.Namespace,
	}
}

func (e *Endpoint) ToProto(ip string) *wv1.Endpoint {
	return &wv1.Endpoint{
		Ip:        ip,
		NodeName:  e.NodeName,
		Kind:      e.Kind,
		Name:      e.Name,
		Namespace: e.Namespace,
	}
}

type Endpoints map[string]Endpoint

func (e Endpoints) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil
	}
	jsonData, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Endpoints to JSON: %w", err)
	}
	return string(jsonData), nil
}

func (e *Endpoints) Scan(value interface{}) error {
	if value == nil {
		*e = nil
		return nil
	}
	var jsonData []byte
	switch v := value.(type) {
	case []byte:
		jsonData = v
	case string:
		jsonData = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Endpoints", value)
	}
	if *e == nil {
		*e = make(Endpoints)
	}
	return json.Unmarshal(jsonData, e)
}

type Upstream struct {
	ID      uint   `gorm:"primaryKey"`
	SvcFqdn string `gorm:"uniqueIndex:idx_svc_fqdn"`
	//ContainerIps      pq.StringArray `gorm:"type:text[]"` // upstream IPs
	Endpoints         Endpoints `gorm:"type:jsonb"`
	UpstreamRouteType uint32
	Ingresses         []Ingress `gorm:"foreignKey:UpstreamID"`
	Ports             []Port    `gorm:"foreignKey:UpstreamID"`
	CreatedAt         time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt         time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type UpstreamRepository struct {
	db       *gorm.DB
	logger   *zap.Logger
	Upstream Upstream
}

func NewUpstreamRepository(tx *gorm.DB, logger *zap.Logger) *UpstreamRepository {
	modelSvc := &UpstreamRepository{db: tx, logger: logger}
	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}
	return modelSvc
}

func NewUpstreamFromRequest(upstreamReq *wv1.Upstream) *Upstream {
	u := &Upstream{
		SvcFqdn:           upstreamReq.SvcFqdn,
		UpstreamRouteType: uint32(upstreamReq.UpstreamRouteType),
		Endpoints:         make(Endpoints),
	}
	// set endpoints
	for _, ep := range upstreamReq.Endpoints {
		u.Endpoints[ep.Ip] = NewEndpointFromRequest(ep)
	}
	return u
}

func (s *UpstreamRepository) Save(u *Upstream) (*Upstream, error) {

	assigmentColumns := []string{
		"endpoints",
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
		Save(&u); res.Error != nil {
		return u, connect.NewError(connect.CodeUnknown, res.Error)
	}
	return u, nil
}

func (s *UpstreamRepository) List(options *wv1.ListRoutesOptions) (upstreams []*Upstream, err error) {
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
		UpstreamRouteType: wv1.UpstreamRouteType(u.UpstreamRouteType),
	}
	if u.Endpoints != nil {
		for ip, ep := range u.Endpoints {
			wv1upstream.Endpoints = append(wv1upstream.Endpoints, ep.ToProto(ip))
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
	currentUpstream := &Upstream{}
	if err := tx.Where("svc_fqdn = ?", u.SvcFqdn).First(currentUpstream).Error; err != nil {
		// set default upstream route type
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	// set default upstream route type
	if u.UpstreamRouteType == uint32(wv1.UpstreamRouteType_UPSTREAM_ROUTE_TYPE_UNSPECIFIED) {
		u.UpstreamRouteType = uint32(wv1.UpstreamRouteType_UPSTREAM_ROUTE_TYPE_PORT)
	}
	return nil
}
