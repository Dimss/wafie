package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/internal/models"
	"go.uber.org/zap"
)

type ProtectionService struct {
	v1.UnimplementedApplicationServiceHandler
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
	l := s.logger.With(zap.Uint32("applicationId", req.Msg.ApplicationId))
	l.Info("creating new protection entry")
	defer l.Info("protection entry created")
	protectionModelSvc := models.NewProtectionModelSvc(nil, l)
	protection, err := protectionModelSvc.CreateProtection(req.Msg)
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
	protectionModelSvc := models.NewProtectionModelSvc(nil, l)
	protection, err := protectionModelSvc.GetProtection(req.Msg)
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
	protectionModelSvc := models.NewProtectionModelSvc(nil, l)
	protection, err := protectionModelSvc.UpdateProtection(req.Msg)
	if err != nil {
		return connect.NewResponse(&cwafv1.PutProtectionResponse{}), err
	}
	return connect.NewResponse(&cwafv1.PutProtectionResponse{
		Protection: protection.ToProto(),
	}), nil
}

func (s *ProtectionService) ListProtections(
	ctx context.Context,
	req *connect.Request[cwafv1.ListProtectionsRequest]) (
	*connect.Response[cwafv1.ListProtectionsResponse], error) {
	s.logger.Info("listing protections")
	defer s.logger.Info("protections listed")
	protectionModelSvc := models.NewProtectionModelSvc(nil, s.logger)
	protections, err := protectionModelSvc.ListProtections(req.Msg.Options)
	if err != nil {
		s.logger.Error("failed to list protections", zap.Error(err))
		return connect.NewResponse(&cwafv1.ListProtectionsResponse{}), err
	}
	var cwafv1Protections []*cwafv1.Protection
	for _, protection := range protections {
		cwafv1Protections = append(cwafv1Protections, protection.ToProto())
	}
	return connect.NewResponse(&cwafv1.ListProtectionsResponse{
		Protections: cwafv1Protections,
	}), nil
}

func (s *ProtectionService) DeleteProtection(
	ctx context.Context,
	req *connect.Request[cwafv1.DeleteProtectionRequest]) (
	*connect.Response[cwafv1.DeleteProtectionResponse], error) {
	l := s.logger.With(zap.Uint32("protectionId", req.Msg.Id))
	l.Info("deleting protection entry")
	defer l.Info("protection entry deleted")
	protectionModelSvc := models.NewProtectionModelSvc(nil, l)
	err := protectionModelSvc.DeleteProtection(req.Msg.Id)
	if err != nil {
		l.Error("failed to delete protection entry", zap.Error(err))
		return connect.NewResponse(&cwafv1.DeleteProtectionResponse{}), err
	}
	return connect.NewResponse(&cwafv1.DeleteProtectionResponse{}), nil
}
