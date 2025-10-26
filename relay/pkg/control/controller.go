package control

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	logger             *zap.Logger
	epsCh              chan *discoveryv1.EndpointSlice
	nodeName           string
	stateVersion       string
	protectionClient   v1.ProtectionServiceClient
	stateVersionClient v1.StateVersionServiceClient
	routeClient        v1.RouteServiceClient
	clientset          *kubernetes.Clientset
}

func NewController(apiAddr, nodeName string, epsCh chan *discoveryv1.EndpointSlice, logger *zap.Logger) (*Controller, error) {
	rc, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	return &Controller{
		logger:   logger,
		epsCh:    epsCh,
		nodeName: nodeName,
		protectionClient: v1.NewProtectionServiceClient(
			http.DefaultClient,
			apiAddr,
		),
		routeClient: v1.NewRouteServiceClient(
			http.DefaultClient,
			apiAddr,
		),
		stateVersionClient: v1.NewStateVersionServiceClient(
			http.DefaultClient, apiAddr,
		),
		clientset: clientset,
	}, nil
}

func (c *Controller) Run() {
	go func() {
		{
			for {
				time.Sleep(1 * time.Second)
				if !c.stateVersionChanged() {
					continue
				}
				mode := wv1.ProtectionMode_PROTECTION_MODE_ON
				includeApps := true
				req := connect.NewRequest(&wv1.ListProtectionsRequest{
					Options: &wv1.ListProtectionsOptions{
						ProtectionMode: &mode,
						IncludeApps:    &includeApps,
					},
				})
				listResp, err := c.protectionClient.ListProtections(context.Background(), req)
				if err != nil {
					c.logger.Error("failed to list protections", zap.Error(err))
					continue
				}
				for _, protection := range listResp.Msg.Protections {
					specs := c.getRelayInstanceSpecs(protection)
					switch protection.ProtectionMode {
					case wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED:
						c.logger.Debug("app protection mode unspecified, skipping")
					case wv1.ProtectionMode_PROTECTION_MODE_ON:
						c.deployRelayInstances(specs)
					case wv1.ProtectionMode_PROTECTION_MODE_OFF:
						c.destroyRelayInstances(specs)
					}
				}

			}
		}
	}()
}

func (c *Controller) stateVersionChanged() bool {
	stateVersionResponse, err := c.stateVersionClient.GetStateVersion(
		context.Background(),
		connect.NewRequest(
			&wv1.GetStateVersionRequest{
				TypeId: wv1.StateTypeId_STATE_TYPE_ID_PROTECTION,
			},
		),
	)
	if err != nil {
		c.logger.Error("failed to get protection state version", zap.Error(err))
		return false
	}
	// check if the protection state has changed since last iteration
	if stateVersionResponse.Msg.StateVersionId == c.stateVersion {
		return false
	}
	c.logger.Info("protection state version has changed",
		zap.String("versionId", stateVersionResponse.Msg.StateVersionId))
	c.stateVersion = stateVersionResponse.Msg.StateVersionId
	return true
}

func (c *Controller) discoverRelayOptions(p *wv1.Protection) (*wv1.RelayOptions, error) {
	if p.ProtectionMode == wv1.ProtectionMode_PROTECTION_MODE_OFF ||
		p.ProtectionMode == wv1.ProtectionMode_PROTECTION_MODE_UNSPECIFIED {
		return &wv1.RelayOptions{}, nil // if protection is off or unspecified, return empty options
	}
	port, err := protectionContainerPort(p)
	if err != nil {
		return &wv1.RelayOptions{}, err
	}
	return &wv1.RelayOptions{
		ProxyFqdn:          "appsecgw.default.svc", // TODO: parameterize this!
		ProxyListeningPort: strconv.Itoa(int(port.ProxyListeningPort)),
		AppContainerPort:   strconv.Itoa(int(port.Number)),
		RelayPort:          "50010", // TODO: currently static-inline, must be configurable
	}, nil
}

func (c *Controller) getRelayInstanceSpecs(protection *wv1.Protection) (rInstances []*RelayInstanceSpec) {
	for _, i := range protection.Application.Ingress {
		for _, ep := range i.Upstream.Endpoints {
			podsClient := c.clientset.CoreV1().Pods(ep.Namespace)
			pod, err := podsClient.Get(context.Background(), ep.Name, metav1.GetOptions{})
			if err != nil {
				c.logger.Error(err.Error())
				continue
			}
			if len(pod.Status.ContainerStatuses) == 0 {
				c.logger.Warn("pod does not contain container status", zap.String("podName", pod.Name))
				continue
			}
			// if current relay instance manager
			// running on different node from the endpoint, skip it
			if ep.NodeName != c.nodeName {
				continue
			}
			// discover relay options
			relayOptions, err := c.discoverRelayOptions(protection)
			if err != nil {
				c.logger.Error("relay options discovery failed", zap.Error(err))
				continue
			}
			i, err := NewRelayInstanceSpec(
				pod.Status.ContainerStatuses[0].ContainerID,
				pod.Name,
				ep.NodeName,
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

func protectionContainerPort(protection *wv1.Protection) (*wv1.Port, error) {
	for _, port := range protection.Application.Ingress[0].Upstream.Ports {
		if port.PortType == wv1.PortType_PORT_TYPE_CONTAINER_PORT {
			return port, nil
		}
	}
	return nil, fmt.Errorf("protectoin [%d] does not have container ports", protection.Id)
}
