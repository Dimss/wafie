package ingresscache

import (
	"fmt"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/internal/logger"
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
		logger: logger.NewLogger().With(zap.String("type", "ingressNormalizer")),
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
	ingressRequest := cwafv1.CreateIngressRequest{Ingress: &cwafv1.Ingress{}}
	if len(ingObj.Spec.Rules) > 0 && len(ingObj.Spec.Rules[0].HTTP.Paths) > 0 {
		ingressRequest.Ingress.Name = fmt.Sprintf("%s.%s.svc", ingObj.Name, ingObj.Namespace)
		ingressRequest.Ingress.Port = 80 // TODO: add support for TLS passthroughs and other protocols later on
		ingressRequest.Ingress.UpstreamPort = ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number
		if ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number == 0 {
			if port, err := getSvcPortByName(
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
				ingObj.Namespace,
			); err != nil {
				i.logger.Error("get service port by name failed", zap.Error(err))
			} else {
				ingressRequest.Ingress.UpstreamPort = port
			}
		}
		ingressRequest.Ingress.Path = ingObj.Spec.Rules[0].HTTP.Paths[0].Path
		ingressRequest.Ingress.Host = ingObj.Spec.Rules[0].Host
		ingressRequest.Ingress.UpstreamHost = ingressRequest.Ingress.Name
		return &ingressRequest, nil
	}
	return nil, nil
}
