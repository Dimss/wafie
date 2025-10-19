package models

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Service struct {
	ID                uint           `gorm:"primaryKey"`
	FQDN              string         `gorm:"uniqueIndex:idx_svc_fqdn"`
	Port              pq.Int32Array  `gorm:"type:integer[]"`
	ContainerPort     pq.Int32Array  `gorm:"type:integer[]"`
	ProxyListenerPort pq.Int32Array  `gorm:"type:integer[]"` // immutable
	UpstreamIps       pq.StringArray `gorm:"type:text[]"`
	// One-to-many relationship
	Ingresses []Ingress `gorm:"foreignKey:ServiceID"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

func (s *Service) allocateProxyListenerPort(tx *gorm.DB) error {

	// TODO: add test for this stuff!
	allocationAttempts := 10
	for allocationAttempts > 0 {
		proxyListenerPort := func() int32 {
			rand.NewSource(time.Now().UnixNano())
			minPort := 49152
			maxPort := 65535
			return int32(rand.Intn(maxPort-minPort) + minPort)
		}()
		ingress := &Ingress{}
		if err := tx.Where("proxy_listener_port = ?", proxyListenerPort).First(ingress).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				s.ProxyListenerPort[0] = proxyListenerPort
				return nil
			}
			return err
		}
		allocationAttempts--
	}
	return fmt.Errorf("error allocating proxy listener port, allocations attempts exceeded")
}

func (s *Service) BeforeCreate(tx *gorm.DB) error {

	//if err := s.allocateProxyListenerPort(tx); err != nil {
	//	return err
	//}
	return nil
}
