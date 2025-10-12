package control

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
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
				specs := c.getRelayInstanceSpecs(eps)
				switch c.appProtectionMode(upstreamHost) {
				case wafiev1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED:
					l.Debug("app protection mode unspecified, skipping")
				case wafiev1.ProtectionMode_PROTECTION_MODE_ON:
					c.deployRelayInstances(specs)
				case wafiev1.ProtectionMode_PROTECTION_MODE_OFF:
					c.destroyRelayInstances(specs)
				}
			}
		}
	}()
}

func (c *Controller) getRelayInstanceSpecs(eps *discoveryv1.EndpointSlice) (rInstances []*RelayInstanceSpec) {
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
		i, err := NewRelayInstanceSpec(pod.Status.ContainerStatuses[0].ContainerID, *ep.NodeName, c.logger)
		if err != nil {
			// TODO: handle an error when container not found due to running on another node
			c.logger.Error(err.Error())
			continue
		}
		rInstances = append(rInstances, i)
	}
	return rInstances
}

func (c *Controller) destroyRelayInstances(relayInstanceSpecs []*RelayInstanceSpec) {
	for _, spec := range relayInstanceSpecs {
		if err := spec.StopSpec(); err != nil {
			c.logger.Error(err.Error())
		}
	}
}

func (c *Controller) deployRelayInstances(relayInstanceSpecs []*RelayInstanceSpec) {
	for _, spec := range relayInstanceSpecs {
		if err := spec.StartSpec(); err != nil {
			c.logger.Error(err.Error())
		}
	}
}

func (c *Controller) appProtectionMode(upstreamHost string) wafiev1.ProtectionMode {
	l := c.logger.With(zap.String("upstreamHost", upstreamHost))
	includeApps := true
	req := connect.NewRequest(&wafiev1.ListProtectionsRequest{
		Options: &wafiev1.ListProtectionsOptions{
			//ProtectionMode: &modeOn,
			//ModSecMode:     &modeOn,
			IncludeApps:  &includeApps,
			UpstreamHost: &upstreamHost,
		},
	})
	protections, err := c.protectionSvcClient.ListProtections(context.Background(), req)
	if err != nil {
		l.Error(fmt.Sprintf("failed to list protections: %v", err))
		return wafiev1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED
	}
	if len(protections.Msg.Protections) == 0 {
		l.Debug("no protections found")
		return wafiev1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED
	}
	l.Debug("protection enabled, relay injection required")
	return protections.Msg.Protections[0].ProtectionMode
}

func upstreamHostFromEndpointSlice(eps *discoveryv1.EndpointSlice) string {
	if eps.ObjectMeta.OwnerReferences != nil &&
		len(eps.ObjectMeta.OwnerReferences) > 0 &&
		eps.ObjectMeta.OwnerReferences[0].Kind == "Service" {
		return fmt.Sprintf("%s.%s.svc", eps.ObjectMeta.OwnerReferences[0].Name, eps.ObjectMeta.Namespace)
	}
	return ""
}
