package ingresscache

import (
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type vs struct{}

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
