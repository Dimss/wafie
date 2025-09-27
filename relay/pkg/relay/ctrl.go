package relay

import (
	"net/http"

	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
)

type Controller struct {
	logger           *zap.Logger
	epsCh            chan *discoveryv1.EndpointSlice
	ingressSvcClient v1.IngressServiceClient
}

func NewController(logger *zap.Logger, epsCh chan *discoveryv1.EndpointSlice, apiAddr string) *Controller {
	return &Controller{
		logger: logger,
		epsCh:  epsCh,
		ingressSvcClient: v1.NewIngressServiceClient(
			http.DefaultClient,
			apiAddr,
		),
	}
}

func (r *Controller) Run() {
	for {
		eps := <-r.epsCh
		r.logger.Info(eps.Name)
	}
}
