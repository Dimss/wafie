package apiserver

import (
	"connectrpc.com/connect"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/models"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type ApplicationService struct {
	cwafv1connect.UnimplementedApplicationServiceHandler
	logger *zap.Logger
	foo    string
}

func NewApplicationService(log *zap.Logger) *ApplicationService {
	return &ApplicationService{
		logger: log,
		foo:    "bar",
	}
}

func (s *ApplicationService) CreateApplication(
	ctx context.Context, req *connect.Request[cwafv1.CreateApplicationRequest]) (
	*connect.Response[cwafv1.CreateApplicationResponse], error) {
	s.logger.With(
		zap.String("name", req.Msg.GetName()),
		zap.String("namespace", req.Msg.GetNamespace())).
		Info("creating new application entry")
	if app, err := models.CreateApplication(req.Msg); err != nil {
		// ToDo: verify if the application already exists
		return connect.NewResponse(&cwafv1.CreateApplicationResponse{}), err
	} else {
		return connect.NewResponse(&cwafv1.CreateApplicationResponse{Id: uint32(app.ID)}), nil
	}
}

func (s *ApplicationService) GetApplication(
	ctx context.Context, req *connect.Request[cwafv1.GetApplicationRequest]) (
	*connect.Response[cwafv1.GetApplicationResponse], error) {
	s.logger.With(
		zap.Uint32("id", req.Msg.GetId())).
		Info("getting application entry")
	app, err := models.GetApplication(req.Msg)
	if err != nil {
		return connect.NewResponse(&cwafv1.GetApplicationResponse{}), err
	}
	return connect.NewResponse(&cwafv1.GetApplicationResponse{
		Application: app.ToProto(),
	}), err
}

func (s *ApplicationService) ListApplications(
	ctx context.Context, req *connect.Request[cwafv1.ListApplicationsRequest]) (
	*connect.Response[cwafv1.ListApplicationsResponse], error) {
	s.logger.Info("start applications listing")
	defer s.logger.Info("end applications listing")
	apps, err := models.ListApplications(req.Msg.Options)
	if err != nil {
		return nil, err
	}
	var cwafv1Apps []*cwafv1.Application
	for _, app := range apps {
		cwafv1Apps = append(cwafv1Apps, app.ToProto())
	}
	return connect.NewResponse(&cwafv1.ListApplicationsResponse{Applications: cwafv1Apps}), nil
}

func (s *ApplicationService) PutApplication(
	ctx context.Context, req *connect.Request[cwafv1.PutApplicationRequest]) (
	*connect.Response[cwafv1.PutApplicationResponse], error) {
	var app *models.Application
	var err error

	if app, err = models.UpdateApplication(req.Msg.Application); err != nil {
		return connect.NewResponse(&cwafv1.PutApplicationResponse{}), err
	}
	return connect.NewResponse(&cwafv1.PutApplicationResponse{
		Application: app.ToProto(),
	}), nil
}
