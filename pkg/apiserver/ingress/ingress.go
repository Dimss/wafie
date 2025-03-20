package ingress

import (
	"connectrpc.com/connect"
	"context"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
)

type Service struct {
	cwafv1connect.UnimplementedIngressServiceHandler
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CreateIngress(
	ctx context.Context,
	req *connect.Request[v1.CreateIngressRequest]) (
	*connect.Response[v1.CreateIngressResponse], error) {
	zap.S().Infof("Name: %s", req.Msg.GetName())
	zap.S().Infof("Namespace: %s", req.Msg.GetNamespace())
	return connect.NewResponse(&v1.CreateIngressResponse{}), nil
}
