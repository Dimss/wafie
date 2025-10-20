package apiserver

import (
	"context"

	"connectrpc.com/connect"
	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/apisrv/internal/models"
	"go.uber.org/zap"
)

type UpstreamService struct {
	wafiev1connect.UnimplementedUpstreamServiceHandler
	logger *zap.Logger
}

func NewUpstreamService(log *zap.Logger) *UpstreamService {
	return &UpstreamService{
		logger: log,
	}
}

func (s *UpstreamService) CreateUpstream(
	ctx context.Context,
	req *connect.Request[cwafv1.CreateUpstreamRequest]) (
	*connect.Response[cwafv1.CreateUpstreamResponse], error) {
	err := models.
		NewUpstreamModelSvc(nil, s.logger).
		Save(
			models.NewUpstreamFromRequest(req.Msg.Upstream),
		)
	return connect.NewResponse(&cwafv1.CreateUpstreamResponse{}), err
}
