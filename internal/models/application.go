package models

import (
	"connectrpc.com/connect"
	"errors"
	"fmt"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"gorm.io/gorm"
	"reflect"
	"time"
)

type Application struct {
	ID          uint         `gorm:"primaryKey"`
	Name        string       `gorm:"uniqueIndex:idx_name_namespace"`
	Namespace   string       `gorm:"uniqueIndex:idx_name_namespace"`
	Protections []Protection `gorm:"foreignKey:ApplicationID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func GetApplication(req *v1.GetApplicationRequest) (*Application, error) {
	app := &Application{
		ID: uint(req.GetId()),
	}
	err := db().Preload("Protections.WAFConfig").First(&app, req.GetId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("application not found"))
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return app, nil
}

func CreateApplication(req *v1.CreateApplicationRequest) (*Application, error) {
	app, err := FromProtoCreateApplicationRequest(req)
	if err != nil {
		return nil, err
	}

	if err := db().Create(app).Error; err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	return app, nil
}

func ListApplications(options *v1.ListApplicationsOptions) ([]*Application, error) {
	var apps []*Application
	whereMap := map[string]interface{}{}
	// ToDo: add filters for application

	res := db().Preload("Protections.WAFConfig").Where(whereMap).Find(&apps)
	if res.Error != nil {
		return nil, connect.NewError(connect.CodeUnknown, res.Error)
	}
	return apps, nil
}

func UpdateApplication(req *v1.Application) (*Application, error) {
	var app Application
	// Prevent changing immutable fields
	if req.GetName() != "" && app.Name != req.GetName() {
		return nil, errors.New("cannot change application name")
	}
	if req.GetNamespace() != "" && app.Namespace != req.GetNamespace() {
		return nil, errors.New("cannot change application namespace")
	}

	// Load app and protections
	if err := db().Preload("Protections.WAFConfig").First(&app, req.GetId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("application not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Build map of existing protections by type
	existingByType := make(map[ProtectionType]*Protection)
	for i := range app.Protections {
		existingByType[app.Protections[i].Type] = &app.Protections[i]
	}

	// Track types sent in update
	seenTypes := make(map[ProtectionType]bool)

	for _, update := range req.GetProtections() {
		if update.Status == nil {
			return nil, errors.New("protection status is required")
		}

		// Determine type based on config
		var pType ProtectionType
		switch update.Config.(type) {
		case *v1.Protection_Waf:
			pType = ProtectionTypeWAF
		default:
			return nil, fmt.Errorf("unsupported or missing protection config")
		}

		seenTypes[pType] = true

		// Update existing or create new
		existing := existingByType[pType]
		if existing != nil {
			// Handle updates
			configChanged := false
			switch cfg := update.Config.(type) {
			case *v1.Protection_Waf:
				if existing.WAFConfig == nil {
					existing.WAFConfig = &WafProtectionConfig{}
					configChanged = true
				}
				if existing.WAFConfig.RuleSet != cfg.Waf.RuleSet {
					existing.WAFConfig.RuleSet = cfg.Waf.RuleSet
					configChanged = true
				}
				if !reflect.DeepEqual(existing.WAFConfig.AllowListIPs, cfg.Waf.AllowListIps) {
					existing.WAFConfig.AllowListIPs = cfg.Waf.AllowListIps
					configChanged = true
				}
			}

			if existing.DesiredState != ProtectionState(update.Status.Desired.String()) {
				configChanged = true
			}

			if configChanged {
				existing.ActualState = ProtectionUnspecified
				existing.LastUpdated = time.Now()
				existing.Reason = "protection updated"
			}

			existing.DesiredState = ProtectionState(update.Status.Desired.String())
			existing.Reason = update.Status.Reason

		} else {
			// Add new protection
			newProtection := Protection{
				ApplicationID: app.ID,
				Type:          pType,
				DesiredState:  ProtectionState(update.Status.Desired.String()),
				ActualState:   ProtectionUnspecified,
				LastUpdated:   time.Now(),
				Reason:        update.Status.Reason,
			}

			switch cfg := update.Config.(type) {
			case *v1.Protection_Waf:
				newProtection.WAFConfig = &WafProtectionConfig{
					RuleSet:      cfg.Waf.RuleSet,
					AllowListIPs: cfg.Waf.AllowListIps,
				}
			}

			app.Protections = append(app.Protections, newProtection)
		}
	}

	// Delete any protections not included in update
	for _, existing := range app.Protections {
		if !seenTypes[existing.Type] {
			if err := db().Where("protection_id = ?", existing.ID).Delete(&WafProtectionConfig{}).Error; err != nil {
				return nil, fmt.Errorf("failed to delete WAF config for protection %d: %w", existing.ID, err)
			}
			if err := db().Delete(&existing).Error; err != nil {
				return nil, fmt.Errorf("failed to delete protection %d: %w", existing.ID, err)
			}
		}
	}

	// Save updated app
	if err := db().Session(&gorm.Session{FullSaveAssociations: true}).Save(&app).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &app, nil
}
