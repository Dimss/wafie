package apiserver

//
//import (
//	"context"
//
//	"connectrpc.com/connect"
//	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
//	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
//	"github.com/Dimss/wafie/apisrv/internal/models"
//	"go.uber.org/zap"
//)
//
//type UpstreamService struct {
//	wafiev1connect.UnimplementedUpstreamServiceHandler
//	logger *zap.Logger
//}
//
//func NewUpstreamService(log *zap.Logger) *UpstreamService {
//	return &UpstreamService{
//		logger: log,
//	}
//}
//
//func (s *UpstreamService) CreateUpstream(
//	ctx context.Context,
//	req *connect.Request[wv1.CreateUpstreamRequest]) (
//	*connect.Response[wv1.CreateUpstreamResponse], error) {
//	err := models.
//		NewUpstreamModelSvc(nil, s.logger).
//		Save(
//			models.NewUpstreamFromRequest(req.Msg.Upstream),
//			req.Msg.Options,
//		)
//	return connect.NewResponse(&wv1.CreateUpstreamResponse{}), err
//}
//
//func (s *UpstreamService) ListUpstreams(
//	ctx context.Context,
//	req *connect.Request[wv1.ListUpstreamsRequest]) (
//	*connect.Response[wv1.ListUpstreamsResponse], error) {
//	upstreams, err := models.
//		NewUpstreamModelSvc(nil, s.logger).
//		List(req.Msg.Options)
//	var upstreamResponse = make([]*wv1.Upstream, len(upstreams))
//	for i, upstream := range upstreams {
//		upstreamResponse[i] = upstream.ToProto()
//	}
//	resp := connect.NewResponse(&wv1.ListUpstreamsResponse{Upstreams: upstreamResponse})
//	if err != nil {
//		return resp,
//			connect.NewError(connect.CodeInternal, err)
//	}
//	return resp, nil
//}
