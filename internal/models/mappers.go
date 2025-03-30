package models

import (
	"errors"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"time"
)

func createFromProtoToApplication(req *v1.CreateApplicationRequest) (*Application, error) {
	if req.Name == "" || req.Namespace == "" {
		return nil, errors.New("name and namespace are required")
	}

	app := &Application{
		Name:      req.Name,
		Namespace: req.Namespace,
	}

	for _, p := range req.Protections {
		protection := Protection{
			Type:         ProtectionType(p.Type.String()),
			DesiredState: ProtectionState(p.Status.Desired.String()),
			ActualState:  ProtectionUnspecified,
			LastUpdated:  time.Now(),
		}

		if p.Type == v1.ProtectionType_WAF && p.GetWaf() != nil {
			waf := p.GetWaf()
			protection.WAFConfig = &WafProtectionConfig{
				RuleSet:      waf.RuleSet,
				AllowListIPs: waf.AllowListIps,
			}
		}

		app.Protections = append(app.Protections, protection)
	}

	return app, nil
}

func ToProtoApplication(app Application) *v1.Application {
	proto := &v1.Application{
		Id:          uint32(app.ID),
		Name:        app.Name,
		Namespace:   app.Namespace,
		Protections: []*v1.Protection{},
	}

	for _, p := range app.Protections {
		if p.Type == ProtectionTypeWAF {
			proto.Protections = append(proto.Protections, &v1.Protection{
				Type: v1.ProtectionType_WAF,
				Status: &v1.ProtectionStatus{
					Desired: v1.ProtectionState(v1.ProtectionState_value[string(p.DesiredState)]),
					Actual:  v1.ProtectionState(v1.ProtectionState_value[string(p.ActualState)]),
					Reason:  p.Reason,
				},
				Config: &v1.Protection_Waf{
					Waf: &v1.WafProtectionConfig{
						RuleSet:      p.WAFConfig.RuleSet,
						AllowListIps: p.WAFConfig.AllowListIPs,
					},
				},
			})
		}
	}

	return proto
}
