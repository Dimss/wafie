package ctrl

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Informer struct {
	logger *zap.Logger
	epsCh  chan *discoveryv1.EndpointSlice
}

func NewInformer(epsCh chan *discoveryv1.EndpointSlice, logger *zap.Logger) *Informer {
	return &Informer{
		epsCh:  epsCh,
		logger: logger,
	}
}

func (i *Informer) Start() {
	go func() {
		var informerStartError error
		i.logger.Info("starting endpoints slice informer")
		for {
			if informerStartError != nil {
				i.logger.Error("informer start error", zap.Error(informerStartError))
				i.logger.Info("restarting informer after error...")
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
							//eps := obj.(*discoveryv1.EndpointSlice)
							i.epsCh <- obj.(*discoveryv1.EndpointSlice)
							//serviceName := eps.Labels["kubernetes.io/service-name"]
							//i.logger.Info("New EndpointSlice",
							//	zap.String("service", serviceName), zap.Int("size", len(eps.Endpoints)))
						},
						UpdateFunc: func(oldObj, newObj interface{}) {
							i.epsCh <- newObj.(*discoveryv1.EndpointSlice)
							//eps := newObj.(*discoveryv1.EndpointSlice)
							//serviceName := eps.Labels["kubernetes.io/service-name"]
							//i.logger.Info("update EndpointSlice",
							//	zap.String("service", serviceName), zap.Int("size", len(eps.Endpoints)))
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
			// Start informer
			informerFactory.Start(ctx.Done())

			// Wait for cache sync
			if !cache.WaitForCacheSync(ctx.Done(), endpointSliceInformer.Informer().HasSynced) {
				fmt.Println("Failed to sync cache")
				cancel()
				continue
			}

			i.logger.Info("EndpointSlice informer running")
			<-ctx.Done()
		}
	}()
}
