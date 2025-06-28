package ingresscache

import (
	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type route struct{}

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
