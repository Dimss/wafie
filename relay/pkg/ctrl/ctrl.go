package ctrl

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
)

type Controller struct {
	logger              *zap.Logger
	epsCh               chan *discoveryv1.EndpointSlice
	protectionSvcClient v1.ProtectionServiceClient
}

func NewController(apiAddr string, epsCh chan *discoveryv1.EndpointSlice, logger *zap.Logger) *Controller {
	return &Controller{
		logger: logger,
		epsCh:  epsCh,
		protectionSvcClient: v1.NewProtectionServiceClient(
			http.DefaultClient,
			apiAddr,
		),
	}
}

func (r *Controller) Run() {
	go func() {
		{
			for {
				eps := <-r.epsCh
				if upstreamHost := svcNameFromEndpointSlice(eps); upstreamHost != "" {
					r.getUpstreamHostIngressSpec(svcNameFromEndpointSlice(eps))
				}

				r.logger.Info(eps.Name)
			}
		}
	}()
}

func (r *Controller) getUpstreamHostIngressSpec(upstreamHost string) {
	l := r.logger.With(zap.String("upstreamHost", upstreamHost))
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
	protections, err := r.protectionSvcClient.ListProtections(context.Background(), req)
	if err != nil {
		l.Error(fmt.Sprintf("failed to list protections: %v", err))
		return
	}
	if len(protections.Msg.Protections) == 0 {
		l.Debug("no protections found")
		return
	}

	l.Info("protection enabled, injecting relay instance...")

}

func svcNameFromEndpointSlice(eps *discoveryv1.EndpointSlice) string {
	if eps.ObjectMeta.OwnerReferences != nil &&
		len(eps.ObjectMeta.OwnerReferences) > 0 &&
		eps.ObjectMeta.OwnerReferences[0].Kind == "Service" {
		return fmt.Sprintf("%s.%s.svc", eps.ObjectMeta.OwnerReferences[0].Name, eps.ObjectMeta.Namespace)
	}
	return ""
}
