package ingresscache

import (
	"fmt"

	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
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

func (i *ingress) normalizedWithError(cwafv1Ing *wafiev1.Ingress, err error) (*wafiev1.Ingress, error) {
	cwafv1Ing.DiscoveryStatus = wafiev1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_INCOMPLETE
	cwafv1Ing.DiscoveryMessage = err.Error()
	return cwafv1Ing, err
}

func (i *ingress) normalize(obj *unstructured.Unstructured) (*wafiev1.Ingress, error) {

	ingObj := &v1.Ingress{}
	cwafv1Ing := &wafiev1.Ingress{}
	if err := runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(obj.Object, ingObj); err != nil {
		return i.normalizedWithError(cwafv1Ing, err)
	}
	objJson, err := obj.MarshalJSON()
	if err != nil {
		return i.normalizedWithError(cwafv1Ing, err)
	}
	if len(ingObj.Spec.Rules) > 0 && len(ingObj.Spec.Rules[0].HTTP.Paths) > 0 {
		if ingObj.Spec.Rules[0].Host == "" {
			i.logger.Info("skipping ingress due to wildcard '*' hostname",
				zap.String("ingress", ingObj.Name+"."+ingObj.Namespace))
			return nil, nil
		}
		//// TODO: fix this!
		//if ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name == controlplane.WafieGatewaySvcName {
		//	i.logger.Info("skipping, ingress already routing to wafie gateway svc",
		//		zap.String("ingress", ingObj.Name+"."+ingObj.Namespace))
		//}
		cwafv1Ing = &wafiev1.Ingress{
			Name:      ingObj.Name,
			Namespace: ingObj.Namespace,
			Port:      80, // TODO: add support for TLS passthroughs and other protocols later on
			UpstreamHost: fmt.Sprintf("%s.%s.svc",
				ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
				ingObj.Namespace),
			Path:           ingObj.Spec.Rules[0].HTTP.Paths[0].Path,
			Host:           ingObj.Spec.Rules[0].Host,
			RawIngressSpec: string(objJson),
			IngressType:    wafiev1.IngressType_INGRESS_TYPE_NGINX,
		}
		cwafv1Ing.UpstreamPort, err = i.discoverUpstreamPort(ingObj)
		if err != nil {
			return i.normalizedWithError(cwafv1Ing, err)
		}
		cwafv1Ing.ContainerPort, err = i.discoverContainerPort(ingObj)
		if err != nil {
			return i.normalizedWithError(cwafv1Ing, err)
		}
		return cwafv1Ing, nil
	}
	return nil, nil
}

func (i *ingress) discoverUpstreamPort(ing *v1.Ingress) (int32, error) {
	if ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != 0 {
		return ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number, nil
	}
	// get service port number by service port name
	if port, err := getSvcPortNumberBySvcPortName(
		ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
		ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
		ing.Namespace,
	); err != nil {
		return 0, err
	} else {
		return port, nil
	}

}

func (i *ingress) discoverContainerPort(ing *v1.Ingress) (int32, error) {
	if ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number == 0 {
		if port, err := getContainerPortBySvcPortName(
			ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
			ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
			ing.Namespace,
		); err != nil {

			return 0, err
		} else {
			return port, nil
		}
	} else {
		if port, err := getContainerPortBySvcPortNumber(
			ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number,
			ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
			ing.Namespace,
		); err != nil {
			return 0, err
		} else {
			return port, nil
		}
	}
}
