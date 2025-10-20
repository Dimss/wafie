package ingress

import (
	"fmt"

	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
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

func (i *ingress) normalizedWithError(upstream *wafiev1.Upstream, err error) (*wafiev1.Upstream, error) {
	if upstream == nil {
		upstream = &wafiev1.Upstream{}
	}
	if upstream.Ingresses == nil || len(upstream.Ingresses) == 0 {
		upstream.Ingresses = []*wafiev1.Ingress{
			{
				DiscoveryStatus:  wafiev1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_INCOMPLETE,
				DiscoveryMessage: err.Error(),
			},
		}
		return upstream, err
	}
	upstream.Ingresses[0].DiscoveryStatus = wafiev1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_INCOMPLETE
	upstream.Ingresses[0].DiscoveryMessage = err.Error()
	return upstream, err
}

func (i *ingress) normalize(obj *unstructured.Unstructured) (upstream *wafiev1.Upstream, err error) {
	upstream = &wafiev1.Upstream{}
	ingObj := &v1.Ingress{}
	if err := runtime.
		DefaultUnstructuredConverter.
		FromUnstructured(obj.Object, ingObj); err != nil {
		return i.normalizedWithError(upstream, err)
	}

	if len(ingObj.Spec.Rules) > 0 && len(ingObj.Spec.Rules[0].HTTP.Paths) > 0 {
		//TODO: check what will happen when the host will be empty, i.e ingress with wildcard scenario
		if ingObj.Spec.Rules[0].Host == "" {
			i.logger.Info("skipping ingress due to wildcard '*' hostname",
				zap.String("ingress", ingObj.Name+"."+ingObj.Namespace))
			return nil, nil
		}
		// set upstream ingress
		upstream.Ingresses = append(upstream.Ingresses,
			&wafiev1.Ingress{
				Name:            ingObj.Name,
				Namespace:       ingObj.Namespace,
				Port:            80, // TODO: add support for TLS passthroughs and other protocols later on
				Path:            ingObj.Spec.Rules[0].HTTP.Paths[0].Path,
				Host:            ingObj.Spec.Rules[0].Host,
				IngressType:     wafiev1.IngressType_INGRESS_TYPE_NGINX,
				DiscoveryStatus: wafiev1.DiscoveryStatusType_DISCOVERY_STATUS_TYPE_SUCCESS,
			},
		)
		upstream.SvcFqdn = fmt.Sprintf("%s.%s.svc",
			ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name,
			ingObj.Namespace)

		upstream.SvcPorts, err = i.discoverSvcPorts(ingObj)
		if err != nil {
			return i.normalizedWithError(upstream, err)
		}
		upstream.ContainerPorts, err = i.discoverContainerPorts(ingObj)
		if err != nil {
			return i.normalizedWithError(upstream, err)
		}
		return upstream, nil
	}
	return nil, nil
}

// discoverSvcPorts is in use when envoy making routing by virtual host
func (i *ingress) discoverSvcPorts(ing *v1.Ingress) (ports []*wafiev1.Port, err error) {
	//
	if ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != 0 {
		return append(ports, &wafiev1.Port{
			Number: uint32(ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number),
			Status: wafiev1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
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
		return append(ports, &wafiev1.Port{
			Number: uint32(port),
			Name:   ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name,
			Status: wafiev1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
		}), nil
	}
}

// discoverContainerPorts in use when envoy making routing by listeners port
func (i *ingress) discoverContainerPorts(ing *v1.Ingress) (ports []*wafiev1.Port, err error) {
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
		return append(ports, &wafiev1.Port{
			Number: uint32(portNumber),
			Name:   portName,
			Status: wafiev1.PortStatusType_PORT_STATUS_TYPE_ENABLED,
		}), nil
	}
}
