package apiserver

import (
	"context"

	"connectrpc.com/connect"

	"buf.build/go/protovalidate"
	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/apisrv/internal/models"
	"go.uber.org/zap"
)

type RouteService struct {
	wafiev1connect.UnimplementedRouteServiceHandler
	logger *zap.Logger
}

func NewRouteService(log *zap.Logger) *RouteService {
	return &RouteService{
		logger: log,
	}
}

func (s *RouteService) CreateRoute(
	ctx context.Context,
	req *connect.Request[wv1.CreateRouteRequest]) (
	*connect.Response[wv1.CreateRouteResponse], error) {
	s.logger.Debug("create route running", zap.String("upstream", req.Msg.Upstream.SvcFqdn))
	if err := protovalidate.Validate(req.Msg); err != nil {
		return connect.NewResponse(&wv1.CreateRouteResponse{}), connect.NewError(connect.CodeInternal, err)
	}
	u, err := models.NewUpstreamModelSvc(nil, s.logger).
		Save(
			models.NewUpstreamFromRequest(req.Msg.Upstream),
			nil,
		)
	if err != nil {
		return connect.NewResponse(&wv1.CreateRouteResponse{}), connect.NewError(connect.CodeInternal, err)
	}
	i := models.NewIngressFromProto(req.Msg.Ingress)
	i.UpstreamID = u.ID // set upstream id
	err = models.NewIngressModelSvc(nil, s.logger).Save(i)
	return connect.NewResponse(&wv1.CreateRouteResponse{}), err
}
