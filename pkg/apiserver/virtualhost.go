package apiserver

import (
	"connectrpc.com/connect"
	"context"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/internal/models"
	"go.uber.org/zap"
)

type VirtualHostService struct {
	wafiev1connect.UnimplementedVirtualHostServiceHandler
	logger *zap.Logger
}

func NewVirtualHostService(log *zap.Logger) *VirtualHostService {
	return &VirtualHostService{
		logger: log,
	}
}

func (s *VirtualHostService) CreateVirtualHost(ctx context.Context,
	req *connect.Request[wafiev1.CreateVirtualHostRequest]) (
	*connect.Response[wafiev1.CreateVirtualHostResponse], error) {
	l := s.logger.With(zap.Uint32("protection_id", req.Msg.ProtectionId))
	l.Info("create virtual host entry")
	defer l.Info("virtual host entry created")
	vh, err := models.NewVirtualHostModelSvc(nil, l).
		CreateVirtualHost(uint(req.Msg.ProtectionId))
	if err != nil {
		l.Error("failed to create virtual host entry", zap.Error(err))
		return connect.NewResponse(&wafiev1.CreateVirtualHostResponse{}), err
	}
	return connect.NewResponse(&wafiev1.CreateVirtualHostResponse{
		Id: uint32(vh.ID),
	}), nil

}

func (s *VirtualHostService) GetVirtualHost(
	ctx context.Context,
	req *connect.Request[wafiev1.GetVirtualHostRequest]) (
	*connect.Response[wafiev1.GetVirtualHostResponse], error) {
	l := s.logger.With(zap.Uint32("id", req.Msg.Id))
	l.Info("getting virtual host entry")
	defer l.Info("virtual host entry retrieved")
	vh, err := models.NewVirtualHostModelSvc(nil, l).
		GetVirtualHostById(uint(req.Msg.Id))
	if err != nil {
		return connect.NewResponse(&wafiev1.GetVirtualHostResponse{}), err
	}
	return connect.NewResponse(
		&wafiev1.GetVirtualHostResponse{
			VirtualHost: vh.ToProto(),
		},
	), nil
}

func (s *VirtualHostService) ListVirtualHosts(
	ctx context.Context,
	req *connect.Request[wafiev1.ListVirtualHostsRequest]) (
	*connect.Response[wafiev1.ListVirtualHostsResponse], error) {
	s.logger.Info("listing virtual host entries")
	defer s.logger.Info("virtual host entries listed")
	vhs, err := models.NewVirtualHostModelSvc(nil, s.logger).
		ListVirtualHosts()
	if err != nil {
		return connect.NewResponse(&wafiev1.ListVirtualHostsResponse{}), err
	}
	virtualHosts := make([]*wafiev1.VirtualHost, len(vhs))
	for i, vh := range vhs {
		virtualHosts[i] = vh.ToProto()
	}
	return connect.NewResponse(
		&wafiev1.ListVirtualHostsResponse{
			VirtualHosts: virtualHosts,
		},
	), nil
}
