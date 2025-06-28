package models

import (
	"connectrpc.com/connect"
	"crypto/md5"
	"encoding/hex"
	"errors"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	assets "github.com/Dimss/wafie/pkg/tmpls"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type VirtualHost struct {
	ID           uint       `gorm:"primaryKey"`
	Spec         string     `gorm:"type:string"`
	SpecChecksum string     `gorm:"type:string"`
	ProtectionID uint       `gorm:"not null;uniqueIndex:idx_virtualhost_protection_id"`
	Protection   Protection `gorm:"foreignKey:ProtectionID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type VirtualHostModelSvc struct {
	db          *gorm.DB
	logger      *zap.Logger
	VirtualHost VirtualHost
}

func NewVirtualHostModelSvc(tx *gorm.DB, logger *zap.Logger) *VirtualHostModelSvc {
	modelSvc := &VirtualHostModelSvc{db: tx, logger: logger}

	if tx == nil {
		modelSvc.db = db()
	}
	if logger == nil {
		modelSvc.logger = applogger.NewLogger()
	}

	return modelSvc

}

func (s *VirtualHostModelSvc) CreateVirtualHost(protectionId uint) (vh *VirtualHost, err error) {

	vh = &VirtualHost{ProtectionID: protectionId}
	protectionModelSvc := NewProtectionModelSvc(s.db, s.logger)
	// get protection
	if protection, err := protectionModelSvc.
		GetProtection(&v1.GetProtectionRequest{Id: uint32(vh.ProtectionID)}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else {
		vh.Protection = *protection
	}
	// parse template
	if vh.Spec, err = vh.parseTemplate(s.db); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// set spec checksum
	vh.specChecksum()
	// save virtual host
	if err := s.db.Create(vh).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("virtual host already exists"))
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return vh, nil
}

func (s *VirtualHostModelSvc) GetVirtualHostById(id uint) (*VirtualHost, error) {
	vh := &VirtualHost{ID: id}
	res := s.db.First(vh)
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("virtual host not found"))
	}
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	return vh, nil
}

func (s *VirtualHostModelSvc) GetVirtualHostByProtectionId(id uint) (*VirtualHost, error) {
	vh := &VirtualHost{ProtectionID: id}
	res := s.db.First(vh)
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("virtual host not found"))
	}
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	return vh, nil
}

func (s *VirtualHostModelSvc) UpdateVirtualHost(id uint) (vh *VirtualHost, err error) {
	vh = &VirtualHost{ID: id}
	//res := s.db.First(vh)
	res := s.db.
		Preload("Protection").
		First(vh)
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("virtual host not found"))
	}
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	// parse template
	if vh.Spec, err = vh.parseTemplate(s.db); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.db.Save(vh).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	return vh, nil
}

func (s *VirtualHostModelSvc) ListVirtualHosts() ([]*VirtualHost, error) {
	var vhs []*VirtualHost
	res := s.db.Find(&vhs)
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	return vhs, nil
}

func (h *VirtualHost) ToProto() *v1.VirtualHost {
	return &v1.VirtualHost{
		Id:       uint32(h.ID),
		Spec:     h.Spec,
		Checksum: h.SpecChecksum,
	}
}

func (h *VirtualHost) loadTemplateData(db *gorm.DB) error {
	if h.Protection.ID == 0 || h.ProtectionID == 0 {
		return errors.New("invalid protection id or protection is not set")
	}
	return db.
		Joins("JOIN applications ON protections.application_id = applications.id").
		Joins("JOIN ingresses on ingresses.application_id = applications.id").
		Preload("Application").
		Preload("Application.Ingress").
		Find(&h.Protection).Error
}

func (h *VirtualHost) parseTemplate(db *gorm.DB) (spec string, err error) {
	if err := h.loadTemplateData(db); err != nil {
		return "", err
	}
	if len(h.Protection.Application.Ingress) == 0 {
		return "", errors.New("no ingress found for application")
	}
	a := assets.NewAssets()
	templateData := map[string]interface{}{
		"ProtectionEnabled": false,
		"UpstreamName":      h.Protection.Application.Name,
		"UpstreamHost":      h.Protection.Application.Ingress[0].UpstreamHost,
		"UpstreamPort":      h.Protection.Application.Ingress[0].UpstreamPort,
		"IngressPort":       80,
		"IngressHost":       h.Protection.Application.Ingress[0].Host,
		"ModSecEnabled":     false,
	}
	if h.Protection.DesiredState.ModSec.Mode == uint32(v1.ProtectionMode_PROTECTION_MODE_ON) {
		templateData["ModSecEnabled"] = true
	}
	if h.Protection.Mode == uint32(v1.ProtectionMode_PROTECTION_MODE_ON) {
		templateData["ProtectionEnabled"] = true
	}
	spec, err = a.RenderVirtualHost(templateData)
	if err != nil {
		return "", err
	}
	return spec, nil
}

func (h *VirtualHost) specChecksum() {
	hash := md5.Sum([]byte(h.Spec))
	h.SpecChecksum = hex.EncodeToString(hash[:])
}

func (h *VirtualHost) BeforeSave(tx *gorm.DB) error {
	// set empty protection, otherwise gorm
	// will try to create all the relations
	// such as application and ingress
	// and will fail b/c they already exists in the db
	h.Protection = Protection{}
	// calculate spec checksum
	h.specChecksum()
	return nil
}
