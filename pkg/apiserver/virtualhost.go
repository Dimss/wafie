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
	vh, err := models.CreateVirtualHost(uint(req.Msg.ProtectionId))
	if err != nil {
		l.Error("failed to create virtual host entry", zap.Error(err))
		return connect.NewResponse(&cwafv1.CreateVirtualHostResponse{}), err
	}
	return connect.NewResponse(&cwafv1.CreateVirtualHostResponse{
		Id: uint32(vh.ID),
	}), nil

}

func (s *VirtualHostService) GetVirtualHost(
	ctx context.Context,
	req *connect.Request[cwafv1.GetVirtualHostRequest]) (
	*connect.Response[cwafv1.GetVirtualHostResponse], error) {
	l := s.logger.With(zap.Uint32("id", req.Msg.Id))
	l.Info("getting virtual host entry")
	defer l.Info("virtual host entry retrieved")
	vh, err := models.GetVirtualHost(uint(req.Msg.Id))
	if err != nil {
		return connect.NewResponse(&cwafv1.GetVirtualHostResponse{}), err
	}
	return connect.NewResponse(
		&cwafv1.GetVirtualHostResponse{
			VirtualHost: vh.ToProto(),
		},
	), nil
}

func (s *VirtualHostService) ListVirtualHosts(
	ctx context.Context,
	req *connect.Request[cwafv1.ListVirtualHostsRequest]) (
	*connect.Response[cwafv1.ListVirtualHostsResponse], error) {
	s.logger.Info("listing virtual host entries")
	defer s.logger.Info("virtual host entries listed")
	vhs, err := models.ListVirtualHosts()
	if err != nil {
		return connect.NewResponse(&cwafv1.ListVirtualHostsResponse{}), err
	}
	virtualHosts := make([]*cwafv1.VirtualHost, len(vhs))
	for i, vh := range vhs {
		virtualHosts[i] = vh.ToProto()
	}
	return connect.NewResponse(
		&cwafv1.ListVirtualHostsResponse{
			VirtualHosts: virtualHosts,
		},
	), nil
}
