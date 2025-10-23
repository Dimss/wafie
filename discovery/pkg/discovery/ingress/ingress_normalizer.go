package ingress

import (
	"fmt"

	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	applogger "github.com/Dimss/wafie/logger"
	"go.uber.org/zap"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func (i *ingress) normalizedWithError(u *wv1.Upstream, ing *wv1.Ingress, err error) (*wv1.Upstream, *wv1.Ingress, error) {
	if u == nil {
		u = &wv1.Upstream{}
	}
	if ing == nil {
		ing = &wv1.Ingress{
			DiscoveryStatus:  wv1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_INCOMPLETE,
			DiscoveryMessage: err.Error(),
		}
	} else {
		ing.DiscoveryStatus = wv1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_INCOMPLETE
		ing.DiscoveryMessage = err.Error()
	}

	return u, nil, err
}

func (i *ingress) normalize(obj *unstructured.Unstructured) (upstream *wv1.Upstream, ingress *wv1.Ingress, err error) {
	upstream = &wv1.Upstream{}
	ingress = &wv1.Ingress{}
	k8sIngress := &v1.Ingress{}
	if err := runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(obj.Object, k8sIngress); err != nil {
		return i.normalizedWithError(upstream, ingress, err)
	}
	if len(k8sIngress.Spec.Rules) > 0 && len(k8sIngress.Spec.Rules[0].HTTP.Paths) > 0 {
		//TODO: check what will happen when the host will be empty, i.e ingress with wildcard scenario
		if k8sIngress.Spec.Rules[0].Host == "" {
			i.logger.Info("skipping ingress due to wildcard '*' hostname",
				zap.String("ingress", k8sIngress.Name+"."+k8sIngress.Namespace))
			return nil, nil, nil
		}
		// set upstream service fqdn
		upstream.SvcFqdn = fmt.Sprintf("%s.%s.svc",
			k8sIngress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
			k8sIngress.Namespace)
		// set upstream services ports
		upstream.SvcPorts, err = i.discoverSvcPorts(k8sIngress)
		if err != nil {
			return i.normalizedWithError(upstream, ingress, err)
		}
		// set upstream containers port
		upstream.ContainerPorts, err = i.discoverContainerPorts(k8sIngress)
		if err != nil {
			return i.normalizedWithError(upstream, ingress, err)
		}
		// set upstream ingress
		ingress = &wv1.Ingress{
			Name:            k8sIngress.Name,
			Namespace:       k8sIngress.Namespace,
			Port:            80, // TODO: add support for TLS passthroughs and other protocols later on
			Path:            k8sIngress.Spec.Rules[0].HTTP.Paths[0].Path,
			Host:            k8sIngress.Spec.Rules[0].Host,
			IngressType:     wv1.IngressType_INGRESS_TYPE_NGINX,
			DiscoveryStatus: wv1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_SUCCESS,
		}
		return upstream, ingress, nil
	}
	return nil, nil, nil
}

// discoverSvcPorts is in use when envoy making routing by virtual host
func (i *ingress) discoverSvcPorts(ing *v1.Ingress) (ports []*wv1.Port, err error) {
	//
	if ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != 0 {
		return append(ports, &wv1.Port{
			Number: uint32(ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number),
			Status: wv1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
		}), nil
	}
	// get service port number by service port name
	if port, err := getSvcPortNumberBySvcPortName(
		ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
		ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
		ing.Namespace,
	); err != nil {
		return nil, err
	} else {
		return append(ports, &wv1.Port{
			Number: uint32(port),
			Name:   ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
			Status: wv1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
		}), nil
	}
}

// discoverContainerPorts in use when envoy making routing by listeners port
func (i *ingress) discoverContainerPorts(ing *v1.Ingress) (ports []*wv1.Port, err error) {
	if portNumber, portName, err := getContainerPortBySvcPort(
		intstr.IntOrString{
			IntVal: ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number,
			StrVal: ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
		},
		ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
		ing.Namespace,
	); err != nil {
		return nil, err
	} else {
		return append(ports, &wv1.Port{
			Number: uint32(portNumber),
			Name:   portName,
			Status: wv1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
		}), nil
	}
}
