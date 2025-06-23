package controlplane

import cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"

func shouldSkipProtection(protection *cwafv1.Protection) bool {
	if protection.Application == nil {
		return true // skip if no application is associated
	}
	if len(protection.Application.Ingress) == 0 {
		return true // skip if no ingress is associated
	}
	return false
}
