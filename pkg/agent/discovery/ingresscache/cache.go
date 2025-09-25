package ingresscache

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	cache2 "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type IngressType = string

const (
	VsIngressType    IngressType = "istio"
	K8sIngressType   IngressType = "ingress"
	RouteIngressType IngressType = "openshift"
)

type normalizer interface {
	gvr() schema.GroupVersionResource
	normalize(*unstructured.Unstructured) (*wafiev1.CreateIngressRequest, error)
}

func newParser(ingressType IngressType) normalizer {
	switch ingressType {
	case K8sIngressType:
		return newIngress()
	case VsIngressType:
		return &vs{}
	case RouteIngressType:
		return &route{}
	}
	zap.S().Fatalf("unknown ingress type: %s", ingressType)
	return nil
}

type IngressCache struct {
	ingressType      IngressType
	normalizer       normalizer
	notifier         chan struct{}
	namespace        string
	ingressSvcClient v1.IngressServiceClient
	logger           *zap.Logger
}

func NewIngressCache(ingressType IngressType, apiAddr string, logger *zap.Logger) *IngressCache {
	cache := &IngressCache{
		ingressType: ingressType,
		notifier:    make(chan struct{}, 1000),
		namespace:   "",
		normalizer:  newParser(ingressType),
		logger:      logger,
		ingressSvcClient: v1.NewIngressServiceClient(
			http.DefaultClient, apiAddr,
		),
	}
	return cache
}

func (c *IngressCache) Start() {

	go func() {
		l := c.logger.With(zap.String("parser", c.ingressType))
		l.Info("starting ingress cache listener")
		var informerStartError error
		for {
			if informerStartError != nil {
				l.Error("informer start error", zap.Error(informerStartError))
				informerStartError = nil
				l.Info("restarting informer after error")
				time.Sleep(3 * time.Second)
			}
			rc, err := config.GetConfig()
			if err != nil {
				informerStartError = err
				continue
			}
			dc, err := dynamic.NewForConfig(rc)
			if err != nil {
				informerStartError = err
				continue
			}
			genericInformer, err := dynamicinformer.NewFilteredDynamicInformer(dc, c.normalizer.gvr(),
				c.namespace, 1*time.Hour, nil, nil), nil
			if err != nil {
				informerStartError = err
				continue
			}
			r, err := genericInformer.Informer().AddEventHandler(cache2.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					unstructuredIngress := obj.(*unstructured.Unstructured)
					if err := c.createIngress(unstructuredIngress); err != nil {
						l.With(
							zap.String("name", unstructuredIngress.GetName()),
							zap.String("namespace", unstructuredIngress.GetNamespace()),
						).Error("error creating ingress", zap.Error(err))
					}
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					l.Info("updated ingress", zap.Any("object", newObj))
				},
				DeleteFunc: func(obj interface{}) {
					l.Info("deleted ingress", zap.Any("object", obj))
				},
			})
			if r.HasSynced() {
			}
			stopCh := make(chan struct{})
			genericInformer.Informer().Run(stopCh)
			<-stopCh
		}
	}()

}

func (c *IngressCache) createIngress(obj *unstructured.Unstructured) error {
	req, err := c.
		normalizer.
		normalize(obj)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	_, err = c.
		ingressSvcClient.
		CreateIngress(
			context.Background(),
			connect.NewRequest(req),
		)
	return err

}
