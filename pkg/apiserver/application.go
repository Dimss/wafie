package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/database"
	"go.uber.org/zap"
)

type ApplicationService struct {
	cwafv1connect.UnimplementedApplicationServiceHandler
}

func NewApplicationService() *ApplicationService {
	return &ApplicationService{}
}

func (s *ApplicationService) CreateApplication(
	ctx context.Context,
	req *connect.Request[cwafv1.CreateApplicationRequest]) (
	*connect.Response[cwafv1.CreateApplicationResponse], error) {
	zap.S().With(
		"name", req.Msg.GetName(),
		"namespace", req.Msg.GetNamespace()).
		Info("creating new application entry")
	if app, err := database.NewApplicationFromRequest(req.Msg); err != nil {
		return connect.NewResponse(&cwafv1.CreateApplicationResponse{}), err
	} else {
		return connect.NewResponse(&cwafv1.CreateApplicationResponse{Id: uint32(app.ID)}), nil
	}
}

func (s *ApplicationService) GetApplication(
	ctx context.Context,
	req *connect.Request[cwafv1.GetApplicationRequest]) (
	*connect.Response[cwafv1.GetApplicationResponse], error) {
	app, err := database.GetApplicationByNameOrId(req.Msg)
	if err != nil {
		return connect.NewResponse(&cwafv1.GetApplicationResponse{}), err
	}
	return connect.NewResponse(&cwafv1.GetApplicationResponse{
		Application: app.ToCwafV1App(),
	}), err
}

func (s *ApplicationService) ListApplications(ctx context.Context,
	req *connect.Request[cwafv1.ListApplicationsRequest]) (
	*connect.Response[cwafv1.ListApplicationResponse], error) {
	apps, err := database.ListApplications(req.Msg.Options)
	if err != nil {
		return nil, err
	}
	var cwafv1Apps []*cwafv1.Application
	for _, app := range apps {
		cwafv1Apps = append(cwafv1Apps, app.ToCwafV1App())
	}
	return connect.NewResponse(&cwafv1.ListApplicationResponse{Applications: cwafv1Apps}), nil
}

func (s *ApplicationService) PutApplication(ctx context.Context,
	req *connect.Request[cwafv1.PutApplicationRequest]) (
	*connect.Response[cwafv1.PutApplicationResponse], error) {
	if err := database.UpdateApplication(req.Msg.Application); err != nil {
		return connect.NewResponse(&cwafv1.PutApplicationResponse{}), err
	}
	return connect.NewResponse(&cwafv1.PutApplicationResponse{}), nil
}
