package endpointslicescache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Cache struct {
	logger                 *zap.Logger
	endpointSliceSvcClient v1.EndpointSliceServiceClient
}

func NewCache(apiAddr string, logger *zap.Logger) *Cache {
	return &Cache{
		logger: logger,
		endpointSliceSvcClient: v1.NewEndpointSliceServiceClient(
			http.DefaultClient,
			apiAddr,
		),
	}
}

func (c *Cache) k8sEndpointSliceToWafieEndpointSlice(eps *discoveryv1.EndpointSlice) (*wafiev1.EndpointSlice, error) {
	var wafieEndpointSlice wafiev1.EndpointSlice
	for _, endpoint := range eps.Endpoints {
		if *endpoint.Conditions.Ready &&
			*endpoint.Conditions.Serving &&
			!*endpoint.Conditions.Terminating {
			if len(endpoint.Addresses) > 0 {
				// TODO: add support for multiple IPs (once some1 will need it)
				wafieEndpointSlice.Ip = endpoint.Addresses[0]
			} else {
				return nil, fmt.Errorf("no IP address in endpoint slice")
			}
			wafieEndpointSlice.NodeName = *endpoint.NodeName
			wafieEndpointSlice.PodUid = string(endpoint.TargetRef.UID)
			wafieEndpointSlice.PodName = endpoint.TargetRef.Name
			wafieEndpointSlice.Namespace = endpoint.TargetRef.Namespace
		}
		b, err := json.Marshal(eps.Ports)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal endpoint slice: %w", err)
		}
		wafieEndpointSlice.Ports = string(b)
	}
	return &wafieEndpointSlice, nil
}

func (c *Cache) saveEndpointSlice(eps *discoveryv1.EndpointSlice) error {
	var wafieEndpointSlice wafiev1.EndpointSlice
	for _, endpoint := range eps.Endpoints {
		if *endpoint.Conditions.Ready &&
			*endpoint.Conditions.Serving &&
			!*endpoint.Conditions.Terminating {

			if len(endpoint.Addresses) > 0 {
				// TODO: add support for multiple IPs (once some1 will need it)
				wafieEndpointSlice.Ip = endpoint.Addresses[0]
			} else {
				c.logger.Error("no IP address in endpoint slice")
				return nil
			}
			wafieEndpointSlice.NodeName = *endpoint.NodeName
			wafieEndpointSlice.PodUid = string(endpoint.TargetRef.UID)
			wafieEndpointSlice.PodName = endpoint.TargetRef.Name
			wafieEndpointSlice.Namespace = endpoint.TargetRef.Namespace
		}
		b, err := json.Marshal(eps.Ports)
		if err != nil {
			c.logger.Error("failed to marshal endpoint slice", zap.Error(err))
			return err
		}
		wafieEndpointSlice.Ports = string(b)

		return err
	}
	return nil
}

func (c *Cache) Start() {
	go func() {
		var informerStartError error
		c.logger.Info("starting endpoints slice informer")
		for {
			if informerStartError != nil {
				c.logger.Error("informer start error", zap.Error(informerStartError))
				c.logger.Info("restarting informer after error...")
				informerStartError = nil
				time.Sleep(3 * time.Second)
			}
			rc, err := config.GetConfig()
			if err != nil {
				informerStartError = err
				continue
			}
			clientset, err := kubernetes.NewForConfig(rc)
			if err != nil {
				informerStartError = err
				continue
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			informerFactory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
			endpointSliceInformer := informerFactory.Discovery().V1().EndpointSlices()
			_, informerStartError = endpointSliceInformer.
				Informer().
				AddEventHandler(
					cache.ResourceEventHandlerFuncs{
						AddFunc: func(obj interface{}) {
							eps := obj.(*discoveryv1.EndpointSlice)
							serviceName := eps.Labels["kubernetes.io/service-name"]
							c.logger.Info("New EndpointSlice",
								zap.String("service", serviceName), zap.Int("size", len(eps.Endpoints)))
						},
						UpdateFunc: func(oldObj, newObj interface{}) {
							eps := newObj.(*discoveryv1.EndpointSlice)
							serviceName := eps.Labels["kubernetes.io/service-name"]
							c.logger.Info("update EndpointSlice",
								zap.String("service", serviceName), zap.Int("size", len(eps.Endpoints)))
						},
						DeleteFunc: func(obj interface{}) {
							// TODO: implement delete logic
							eps := obj.(*discoveryv1.EndpointSlice)
							serviceName := eps.Labels["kubernetes.io/service-name"]
							fmt.Printf("EndpointSlice deleted for service %s\n", serviceName)
						},
					},
				)
			// make sure handlers successfully added
			if informerStartError != nil {
				cancel()
				continue
			}
			// StartSpec informer
			informerFactory.Start(ctx.Done())

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), endpointSliceInformer.Informer().HasSynced) {
				fmt.Println("Failed to sync cache")
				cancel()
				continue
			}

			fmt.Println("EndpointSlice informer running...")
			<-ctx.Done()
		}
	}()
}
