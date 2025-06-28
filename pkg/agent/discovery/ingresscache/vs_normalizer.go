package ingresscache

import (
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
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
func (s *vs) normalize(obj *unstructured.Unstructured) (*wafiev1.CreateIngressRequest, error) {
	return nil, nil
}
