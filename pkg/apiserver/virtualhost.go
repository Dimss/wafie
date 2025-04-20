package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/models"
	"go.uber.org/zap"
)

type VirtualHostService struct {
	cwafv1connect.UnimplementedVirtualHostServiceHandler
	logger *zap.Logger
}

func NewVirtualHostService(log *zap.Logger) *VirtualHostService {
	return &VirtualHostService{
		logger: log,
	}
}

func (s *VirtualHostService) CreateVirtualHost(ctx context.Context,
	req *connect.Request[cwafv1.CreateVirtualHostRequest]) (
	*connect.Response[cwafv1.CreateVirtualHostResponse], error) {
	l := s.logger.With(zap.Uint32("protection_id", req.Msg.ProtectionId))
	l.Info("create virtual host entry")
	defer l.Info("virtual host entry created")
	models.CreateVirtualHost(uint(req.Msg.ProtectionId))
	//models.ParseTemplate(uint(req.Msg.ProtectionId))
	return nil, nil

}

func (s *VirtualHostService) GetVirtualHost(
	ctx context.Context,
	req *connect.Request[cwafv1.GetVirtualHostRequest]) (
	*connect.Response[cwafv1.GetVirtualHostResponse], error) {
	l := s.logger.With(zap.Uint32("protection_id", req.Msg.ProtectionId))
	l.Info("getting virtual host entry")
	defer l.Info("virtual host entry retrieved")
	return nil, nil
	//virtualHost, err := models.GetVirtualHost(req.Msg)
	//if err != nil {
	//	l.Error("failed to get virtual host entry", zap.Error(err))
	//	return connect.NewResponse(&cwafv1.GetVirtualHostResponse{}), err
	//}
	//return connect.NewResponse(&cwafv1.GetVirtualHostResponse{
	//	VirtualHost: virtualHost.ToProto(),
	//}), nil
}
