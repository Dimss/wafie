package models

import (
	"errors"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	assets "github.com/Dimss/cwaf/pkg/tmpls"
	"time"
)

type VirtualHost struct {
	ID           uint       `gorm:"primaryKey"`
	Spec         string     `gorm:"type:string"`
	ProtectionID uint       `gorm:"not null;uniqueIndex:idx_virtualhost_protection_id"`
	Protection   Protection `gorm:"foreignKey:ProtectionID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func CreateVirtualHost(protectionId uint) (*VirtualHost, error) {

	vh := &VirtualHost{ProtectionID: protectionId}
	// get protection
	if protection, err := GetProtection(&v1.GetProtectionRequest{Id: uint32(vh.ProtectionID)}); err != nil {
		return nil, err
	} else {
		vh.Protection = *protection
	}
	// parse template
	if err := vh.parseTemplate(); err != nil {
		return nil, err
	}
	return nil, nil
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
		return nil
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
	h.Spec, err = a.RenderVirtualHost(templateData)
	if err != nil {
		return err
	}
	return nil
}
