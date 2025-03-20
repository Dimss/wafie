package ingresscache

import (
	"fmt"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	cache2 "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"
	"time"
)

type IngressType = string

const (
	VsIngressType    IngressType = "istio"
	K8sIngressType   IngressType = "ingress"
	RouteIngressType IngressType = "openshift"
)

type parser interface {
	gvr() schema.GroupVersionResource
	//parse(*unstructured.Unstructured) *DataCache
}

func newParser(ingressType IngressType) parser {
	switch ingressType {
	case K8sIngressType:
		return &ingress{}
	case VsIngressType:
		return &vs{}
	case RouteIngressType:
		return &route{}
	}
	zap.S().Fatalf("unknown ingress type: %s", ingressType)
	return nil
}

type IngressCache struct {
	ingressType IngressType
	parser      parser
	//cacheChUpdater chan *DataCache
	cacheStore *sync.Map
	notifier   chan struct{}
	namespace  string
}

//var ingressCache *IngressCache

func NewIngressCache(ingressType IngressType, ns string) *IngressCache {
	//if ingressCache != nil {
	//	return ingressCache
	//}
	cache := &IngressCache{
		ingressType: ingressType,
		//cacheChUpdater: make(chan *DataCache, 1000),
		cacheStore: &sync.Map{},
		notifier:   make(chan struct{}, 1000),
		namespace:  ns,
		parser:     newParser(ingressType),
	}

	// start cache updater
	//cache.startCacheUpdater()

	// start informer
	//cache.Start()

	// set global variable
	//ingressCache = cache

	return cache
}

func (c *IngressCache) log() *zap.SugaredLogger {
	return zap.S().With("parser", c.ingressType)
}

func (c *IngressCache) Start() {

	go func() {
		l := c.log()
		var informerStartError error
		for {

			if informerStartError != nil {
				l.Error(informerStartError)
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

			// about informer period: https://groups.google.com/g/kubernetes-sig-api-machinery/c/PbSCXdLDno0
			genericInformer, err := dynamicinformer.NewFilteredDynamicInformer(dc, c.parser.gvr(),
				c.namespace, 1*time.Hour, nil, nil), nil
			if err != nil {
				informerStartError = err
				continue
			}
			r, err := genericInformer.Informer().AddEventHandler(cache2.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					obj.(*unstructured.Unstructured).GetName()
					obj.(*unstructured.Unstructured).GetNamespace()
					c.log().Info(fmt.Sprintf("Added ingress object: %v", obj))
					//c.cacheChUpdater <- c.parser.parse(obj.(*unstructured.Unstructured))
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					c.log().Infof("updated ingress object: %v", newObj)
					//c.cacheChUpdater <- c.parser.parse(newObj.(*unstructured.Unstructured))
				},
				DeleteFunc: func(obj interface{}) {
					c.log().Infof("deleted ingress object: %v", obj)
					//if dataCache := c.parser.parse(obj.(*unstructured.Unstructured)); dataCache != nil {
					//	c.cacheStore.Delete(dataCache.Host)
					//}
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

//func (c *IngressCache) startCacheUpdater() {
//	go func() {
//		l := c.log()
//		for dc := range c.cacheChUpdater {
//			if dc != nil {
//				l.
//					With("host", dc.Host,
//						"authEnabled", dc.SsoEnabled,
//						"skipAuthRoutesTotal", len(dc.SkipAuthRoutes)).
//					Info("ingress cache updated")
//				c.cacheStore.Store(dc.Host, dc)
//				c.notifier <- struct{}{}
//			}
//		}
//	}()
//}

//
//func (c *IngressCache) HostDataCache(host string) *DataCache {
//	if dc, ok := c.cacheStore.Load(host); ok {
//		return dc.(*DataCache)
//	}
//	return &DataCache{}
//
//}
//
//func (c *IngressCache) CacheStore() *sync.Map {
//	return c.cacheStore
//}
//
//func (c *IngressCache) Notifier() chan struct{} {
//	return c.notifier
//}
