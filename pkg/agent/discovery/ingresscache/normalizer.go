package ingresscache

import (
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ingress struct{}
type route struct{}
type vs struct{}

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
		ingressRequest.Ingress.Name = ingObj.Name
		ingressRequest.Ingress.Namespace = ingObj.Namespace
		ingressRequest.Ingress.PortNumber = ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number
		ingressRequest.Ingress.PortName = ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name
		ingressRequest.Ingress.Path = ingObj.Spec.Rules[0].HTTP.Paths[0].Path
		ingressRequest.Ingress.Host = ingObj.Spec.Rules[0].Host
		ingressRequest.Ingress.ServiceName = ingObj.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name
		return &ingressRequest, nil
	}

	return nil, nil

}

func (s *vs) gvr() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}
}
func (s *vs) normalize(obj *unstructured.Unstructured) (*cwafv1.CreateIngressRequest, error) {
	return nil, nil
}

func (r *route) gvr() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}
}
func (r *route) normalize(obj *unstructured.Unstructured) (*cwafv1.CreateIngressRequest, error) {
	return nil, nil
}
