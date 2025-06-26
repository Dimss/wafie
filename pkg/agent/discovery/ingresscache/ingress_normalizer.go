package ingresscache

import (
	"fmt"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/applogger"
	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ingress struct {
	logger *zap.Logger
}

func newIngress() *ingress {
	return &ingress{
		logger: applogger.NewLogger().With(zap.String("type", "ingressNormalizer")),
	}
}

func (i *ingress) gvr() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}
}
func (i *ingress) normalize(obj *unstructured.Unstructured) (*cwafv1.CreateIngressRequest, error) {

	ingObj := &v1.Ingress{}
	if err := runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(obj.Object, ingObj); err != nil {
		return nil, err
	}
	objJson, err := obj.MarshalJSON()
	if err != nil {
		i.logger.Error("failed to marshal ingress object to JSON", zap.Error(err))
		return nil, err
	}
	if len(ingObj.Spec.Rules) > 0 && len(ingObj.Spec.Rules[0].HTTP.Paths) > 0 {
		if ingObj.Spec.Rules[0].Host == "" {
			i.logger.Info("skipping ingress due to wildcard '*' hostname",
				zap.String("ingress", ingObj.Name+"."+ingObj.Namespace))
			return nil, nil
		}
		cwafv1Ing := &cwafv1.Ingress{
			Name:         ingObj.Name,
			Namespace:    ingObj.Namespace,
			Port:         80, // TODO: add support for TLS passthroughs and other protocols later on
			UpstreamPort: ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number,
			UpstreamHost: fmt.Sprintf("%s.%s.svc",
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
				ingObj.Namespace),
			Path:           ingObj.Spec.Rules[0].HTTP.Paths[0].Path,
			Host:           ingObj.Spec.Rules[0].Host,
			RawIngressSpec: string(objJson),
			IngressType:    cwafv1.IngressType_INGRESS_TYPE_NGINX,
		}
		if ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number == 0 {
			if port, err := getSvcPortByName(
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
				ingObj.Namespace,
			); err != nil {
				i.logger.Error("get service port by name failed", zap.Error(err))
			} else {
				cwafv1Ing.UpstreamPort = port
			}
		}
		return &cwafv1.CreateIngressRequest{Ingress: cwafv1Ing}, nil
	}
	return nil, nil
}
