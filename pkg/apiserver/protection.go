package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/models"

	"go.uber.org/zap"
)

type ProtectionService struct {
	cwafv1connect.UnimplementedApplicationServiceHandler
	logger *zap.Logger
}

func NewProtectionService(log *zap.Logger) *ProtectionService {
	return &ProtectionService{
		logger: log,
	}
}

func (s *ProtectionService) CreateProtection(
	ctx context.Context,
	req *connect.Request[cwafv1.CreateProtectionRequest]) (
	*connect.Response[cwafv1.CreateProtectionResponse], error) {
	l := zap.S().With("applicationId", req.Msg.Protection.ApplicationId)
	l.Info("creating new protection entry")
	defer l.Info("protection entry created")
	protection, err := models.CreateProtection(req.Msg)
	if err != nil {
		l.Error("failed to create protection entry", zap.Error(err))
		return connect.NewResponse(&cwafv1.CreateProtectionResponse{}), connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&cwafv1.CreateProtectionResponse{Id: uint32(protection.ID)}), nil
}
