package ctrl

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Controller is responsible for manging a lifecycle (start,stop,restart) of relay instances
type Controller struct {
	logger              *zap.Logger
	epsCh               chan *discoveryv1.EndpointSlice
	protectionSvcClient v1.ProtectionServiceClient
	clientset           *kubernetes.Clientset
}

func NewController(apiAddr string, epsCh chan *discoveryv1.EndpointSlice, logger *zap.Logger) (*Controller, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	return &Controller{
		logger: logger,
		epsCh:  epsCh,
		protectionSvcClient: v1.NewProtectionServiceClient(
			http.DefaultClient,
			apiAddr,
		),
		clientset: clientset,
	}, nil
}

func (c *Controller) Run() {
	go func() {
		{
			for {
				eps := <-c.epsCh
				l := c.logger.With(zap.String("endpointSliceName", eps.Name))
				l.Debug("received endpoint slice")
				// get upstream host from endpoint slice
				upstreamHost := upstreamHostFromEndpointSlice(eps)
				if upstreamHost == "" {
					l.Debug("upstream host is empty")
					continue
				}
				// check if protection enabled for given upstream host
				if !c.protectionRequired(upstreamHost) {
					l.Debug("upstream host protection is off")
					continue
				}
				// inject relay instance
				c.injectRelayInstance(eps)
			}
		}
	}()
}

func (c *Controller) injectRelayInstance(eps *discoveryv1.EndpointSlice) {
	podsClient := c.clientset.CoreV1().Pods(eps.Namespace)
	for _, ep := range eps.Endpoints {
		pod, err := podsClient.Get(context.Background(), ep.TargetRef.Name, metav1.GetOptions{})
		if err != nil {
			c.logger.Error(err.Error())
			continue
		}
		if len(pod.Status.ContainerStatuses) == 0 {
			c.logger.Warn("pod does not contain container status", zap.String("podName", pod.Name))
			continue
		}
		i, err := relay.NewInjector(pod.Status.ContainerStatuses[0].ContainerID, *ep.NodeName, c.logger)
		if err != nil {
			c.logger.Error(err.Error())
			continue
		}
		if err := i.Start(); err != nil {
			c.logger.Error(err.Error())
		}
	}
}

func (c *Controller) protectionRequired(upstreamHost string) bool {
	l := c.logger.With(zap.String("upstreamHost", upstreamHost))
	includeApps := true
	modeOn := wafiev1.ProtectionMode_PROTECTION_MODE_ON
	req := connect.NewRequest(&wafiev1.ListProtectionsRequest{
		Options: &wafiev1.ListProtectionsOptions{
			ProtectionMode: &modeOn,
			ModSecMode:     &modeOn,
			IncludeApps:    &includeApps,
			UpstreamHost:   &upstreamHost,
		},
	})
	protections, err := c.protectionSvcClient.ListProtections(context.Background(), req)
	if err != nil {
		l.Error(fmt.Sprintf("failed to list protections: %v", err))
		return false
	}
	if len(protections.Msg.Protections) == 0 {
		l.Debug("no protections found")
		return false
	}
	l.Debug("protection enabled, injecting relay is needed")
	return true
}

func upstreamHostFromEndpointSlice(eps *discoveryv1.EndpointSlice) string {
	if eps.ObjectMeta.OwnerReferences != nil &&
		len(eps.ObjectMeta.OwnerReferences) > 0 &&
		eps.ObjectMeta.OwnerReferences[0].Kind == "Service" {
		return fmt.Sprintf("%s.%s.svc", eps.ObjectMeta.OwnerReferences[0].Name, eps.ObjectMeta.Namespace)
	}
	return ""
}
