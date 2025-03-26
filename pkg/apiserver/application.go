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
	return connect.NewResponse(&cwafv1.GetApplicationResponse{
		Application: &cwafv1.Application{
			Id:        uint32(app.ID),
			Name:      app.Name,
			Namespace: app.Namespace,
			Protected: app.Protected,
		},
	}), err
}

func (s *ApplicationService) ListApplications(ctx context.Context,
	req *connect.Request[cwafv1.ListApplicationsRequest]) (
	*connect.Response[cwafv1.ListApplicationResponse], error) {

	return nil, nil
}
