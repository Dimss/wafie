package control

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Controller is responsible for manging a lifecycle (start,stop,restart) of relay instances
type Controller struct {
	logger           *zap.Logger
	epsCh            chan *discoveryv1.EndpointSlice
	protectionClient v1.ProtectionServiceClient
	routeClient      v1.RouteServiceClient
	clientset        *kubernetes.Clientset
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
		protectionClient: v1.NewProtectionServiceClient(
			http.DefaultClient,
			apiAddr,
		),
		routeClient: v1.NewRouteServiceClient(
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
				l := c.logger.With(zap.String("endpointslicesname", eps.Name))
				l.Debug("received endpoint slice")
				// get upstream host from endpoint slice
				upstreamHost := upstreamHostFromEndpointSlice(eps)
				if upstreamHost == "" {
					l.Debug("upstream host is empty")
					continue
				}
				protection := c.appProtection(upstreamHost)
				specs := c.getRelayInstanceSpecs(eps, protection)
				switch protection.ProtectionMode {
				case wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED:
					l.Debug("app protection mode unspecified, skipping")
				case wv1.ProtectionMode_PROTECTION_MODE_ON:
					c.deployRelayInstances(specs)
					//case wv1.ProtectionMode_PROTECTION_MODE_OFF:
					//	c.destroyRelayInstances(specs)
				}
			}
		}
	}()
}

func (c *Controller) discoverRelayOptions(p *wv1.Protection) (*wv1.RelayOptions, error) {
	if p.ProtectionMode == wv1.ProtectionMode_PROTECTION_MODE_OFF ||
		p.ProtectionMode == wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED {
		return &wv1.RelayOptions{}, nil // if protection is off or unspecified, return empty options
	}
	// TODO: make the AppSecGW svc fqdn configurable
	appGwSvcFqdn := "wafie-control-plane.default.svc"
	resp, err := c.routeClient.ListRoutes(context.Background(),
		connect.NewRequest(
			&wv1.ListRoutesRequest{
				Options: &wv1.ListRoutesOptions{
					SvcFqdn: &appGwSvcFqdn,
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	upstreams := resp.Msg.Upstreams
	if len(upstreams) == 0 {
		return nil, fmt.Errorf("can not detect proxy ips")
	}

	return &wv1.RelayOptions{
		ProxyIps: upstreams[0].ContainerIps,
		ProxyListeningPort: strconv.
			Itoa(
				int(
					p.Application.Ingress[0].Upstream.ContainerPorts[0].ProxyListeningPort),
			),
		AppContainerPort: strconv.
			Itoa(
				int(p.Application.Ingress[0].Upstream.ContainerPorts[0].Number),
			),
		RelayPort: "50010",
	}, nil
}

func (c *Controller) getRelayInstanceSpecs(eps *discoveryv1.EndpointSlice, protection *wv1.Protection) (rInstances []*RelayInstanceSpec) {
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
		relayOptions, err := c.discoverRelayOptions(protection)
		if err != nil {
			c.logger.Error(err.Error())
			continue
		}
		i, err := NewRelayInstanceSpec(
			pod.Status.ContainerStatuses[0].ContainerID,
			pod.Name,
			*ep.NodeName,
			relayOptions,
			c.logger,
		)
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
			c.logger.Error(err.Error(), zap.String("podName", spec.podName))
		}
	}
}

func (c *Controller) appProtection(upstreamHost string) *wv1.Protection {
	l := c.logger.With(zap.String("upstreamHost", upstreamHost))
	includeApps := true
	req := connect.NewRequest(&wv1.ListProtectionsRequest{
		Options: &wv1.ListProtectionsOptions{
			//ProtectionMode: &modeOn,
			//ModSecMode:     &modeOn,
			IncludeApps:  &includeApps,
			UpstreamHost: &upstreamHost,
		},
	})
	protections, err := c.protectionClient.ListProtections(context.Background(), req)
	if err != nil {
		l.Error(fmt.Sprintf("failed to list protections: %v", err))
		return &wv1.Protection{ProtectionMode: wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED}
	}
	if len(protections.Msg.Protections) == 0 {
		l.Debug("no protections found")
		return &wv1.Protection{ProtectionMode: wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED}
	}
	l.Debug("protection enabled, relay injection required")
	return protections.Msg.Protections[0]
}

func upstreamHostFromEndpointSlice(eps *discoveryv1.EndpointSlice) string {
	if eps.ObjectMeta.OwnerReferences != nil &&
		len(eps.ObjectMeta.OwnerReferences) > 0 &&
		eps.ObjectMeta.OwnerReferences[0].Kind == "Service" {
		return fmt.Sprintf("%s.%s.svc", eps.ObjectMeta.OwnerReferences[0].Name, eps.ObjectMeta.Namespace)
	}
	return ""
}
