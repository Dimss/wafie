package models

import (
	"errors"
	"fmt"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"time"
)

func ToProtoApplication(app Application) *v1.Application {
	proto := &v1.Application{
		Id:          uint32(app.ID),
		Name:        app.Name,
		Namespace:   app.Namespace,
		Protections: []*v1.Protection{},
	}

	for _, p := range app.Protections {
		status := &v1.ProtectionStatus{
			Desired: v1.ProtectionState(v1.ProtectionState_value[string(p.DesiredState)]),
			Actual:  v1.ProtectionState(v1.ProtectionState_value[string(p.ActualState)]),
			Reason:  p.Reason,
		}

		// Build Protection with config inline
		switch p.Type {
		case ProtectionTypeWAF:
			if p.WAFConfig != nil {
				proto.Protections = append(proto.Protections, &v1.Protection{
					Status: status,
					Config: &v1.Protection_Waf{
						Waf: &v1.WafProtectionConfig{
							RuleSet:      p.WAFConfig.RuleSet,
							AllowListIps: p.WAFConfig.AllowListIPs,
						},
					},
				})
			}
		default:
			// Skip unsupported/unknown types
			continue
		}
	}

	return proto
}

func FromProtoCreateApplicationRequest(req *v1.CreateApplicationRequest) (*Application, error) {
	if req.Name == "" || req.Namespace == "" {
		return nil, errors.New("name and namespace are required")
	}

	app := &Application{
		Name:        req.Name,
		Namespace:   req.Namespace,
		Protections: []Protection{},
	}

	for _, p := range req.Protections {
		if p.Status == nil {
			return nil, errors.New("protection status is required")
		}

		protection := Protection{
			DesiredState: ProtectionState(p.Status.Desired.String()),
			ActualState:  ProtectionUnspecified,
			LastUpdated:  time.Now(),
		}

		switch cfg := p.Config.(type) {
		case *v1.CreateProtection_Waf:
			protection.Type = ProtectionTypeWAF
			protection.WAFConfig = &WafProtectionConfig{
				RuleSet:      cfg.Waf.RuleSet,
				AllowListIPs: cfg.Waf.AllowListIps,
			}
		default:
			return nil, fmt.Errorf("unsupported or missing protection config")
		}

		app.Protections = append(app.Protections, protection)
	}

	return app, nil
}
