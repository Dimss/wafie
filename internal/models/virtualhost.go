package models

import (
	"connectrpc.com/connect"
	"crypto/md5"
	"encoding/hex"
	"errors"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	assets "github.com/Dimss/cwaf/pkg/tmpls"
	"github.com/jackc/pgx/v5/pgconn"
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

func CreateVirtualHost(protectionId uint) (*VirtualHost, error) {

	vh := &VirtualHost{ProtectionID: protectionId}
	// get protection
	if protection, err := GetProtection(&v1.GetProtectionRequest{Id: uint32(vh.ProtectionID)}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else {
		vh.Protection = *protection
	}
	// parse template
	if err := vh.parseTemplate(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// set spec checksum
	vh.specChecksum()
	// save virtual host
	if err := db().Create(vh).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("virtual host already exists"))
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return vh, nil
}

func GetVirtualHost(id uint) (*VirtualHost, error) {
	vh := &VirtualHost{ID: id}
	res := db().First(vh)
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("virtual host not found"))
	}
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, res.Error)
	}
	return vh, nil
}

func ListVirtualHosts() ([]*VirtualHost, error) {
	var vhs []*VirtualHost
	res := db().Find(&vhs)
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

func (h *VirtualHost) loadTemplateData() error {
	if h.Protection.ID == 0 || h.ProtectionID == 0 {
		return errors.New("invalid protection id")
	}
	return db().
		Joins("JOIN applications ON protections.application_id = applications.id").
		Joins("JOIN ingresses on ingresses.application_id = applications.id").
		Preload("Application").
		Preload("Application.Ingress").
		Find(&h.Protection).Error
}

func (h *VirtualHost) parseTemplate() (err error) {
	if err := h.loadTemplateData(); err != nil {
		return err
	}
	if len(h.Protection.Application.Ingress) == 0 {
		return errors.New("no ingress found for application")
	}
	a := assets.NewAssets()
	templateData := map[string]interface{}{
		"UpstreamName":  h.Protection.Application.Name,
		"UpstreamHost":  h.Protection.Application.Ingress[0].UpstreamHost,
		"UpstreamPort":  h.Protection.Application.Ingress[0].UpstreamPort,
		"IngressPort":   80,
		"IngressHost":   h.Protection.Application.Ingress[0].Host,
		"ModSecEnabled": false,
	}
	if h.Protection.DesiredState.ModSec.Mode == uint32(v1.ProtectionMode_PROTECTION_MODE_ON) {
		templateData["ModSecEnabled"] = true
	}
	h.Spec, err = a.RenderVirtualHost(templateData)
	if err != nil {
		return err
	}
	return nil
}

func (h *VirtualHost) specChecksum() {
	hash := md5.Sum([]byte(h.Spec))
	h.SpecChecksum = hex.EncodeToString(hash[:])
}
