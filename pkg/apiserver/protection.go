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
	l := s.logger.With(zap.Uint32("applicationId", req.Msg.Protection.ApplicationId))
	l.Info("creating new protection entry")
	defer l.Info("protection entry created")
	protection, err := models.CreateProtection(req.Msg)
	if err != nil {
		l.Error("failed to create protection entry", zap.Error(err))
		return connect.NewResponse(&cwafv1.CreateProtectionResponse{}), connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&cwafv1.CreateProtectionResponse{
		Protection: protection.ToProto(),
	}), nil
}

func (s *ProtectionService) GetProtection(
	ctx context.Context,
	req *connect.Request[cwafv1.GetProtectionRequest]) (
	*connect.Response[cwafv1.GetProtectionResponse], error) {
	l := s.logger.With(zap.Uint32("protectionId", req.Msg.Id))
	l.Info("getting protection entry")
	defer l.Info("protection entry retrieved")
	protection, err := models.GetProtection(req.Msg)
	if err != nil {
		l.Error("failed to get protection entry", zap.Error(err))
		return connect.NewResponse(&cwafv1.GetProtectionResponse{}), err
	}
	return connect.NewResponse(&cwafv1.GetProtectionResponse{
		Protection: protection.ToProto(),
	}), nil
}

func (s *ProtectionService) PutProtection(
	ctx context.Context,
	req *connect.Request[cwafv1.PutProtectionRequest]) (
	*connect.Response[cwafv1.PutProtectionResponse], error) {
	l := s.logger.With(zap.Uint32("protectionId", req.Msg.Id))
	l.Info("updating protection entry")
	defer l.Info("protection entry updated")
	protection, err := models.UpdateProtection(req.Msg)
	if err != nil {
		return connect.NewResponse(&cwafv1.PutProtectionResponse{}), err
	}
	return connect.NewResponse(&cwafv1.PutProtectionResponse{
		Protection: protection.ToProto(),
	}), nil

}
