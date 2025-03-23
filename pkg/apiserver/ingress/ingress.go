package ingress

import (
	"connectrpc.com/connect"
	"context"
	v1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/database/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	cwafv1connect.UnimplementedIngressServiceHandler
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) CreateIngress(
	ctx context.Context,
	req *connect.Request[v1.CreateIngressRequest]) (
	*connect.Response[v1.CreateIngressResponse], error) {
	zap.S().With(
		"name", req.Msg.GetName(),
		"namespace", req.Msg.GetNamespace()).
		Info("creating new ingress entry")
	dbRes := s.db.Create(model.NewIngressFromRequest(req.Msg))
	return connect.NewResponse(&v1.CreateIngressResponse{}), dbRes.Error
}
