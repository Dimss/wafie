package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/database"
	"go.uber.org/zap"
)

type IngressService struct {
	cwafv1connect.UnimplementedIngressServiceHandler
	appSvc *ApplicationService
}

func NewIngressService() *IngressService {
	return &IngressService{
		appSvc: NewApplicationService(),
	}
}

func (s *IngressService) CreateIngress(
	ctx context.Context,
	req *connect.Request[cwafv1.CreateIngressRequest]) (
	*connect.Response[cwafv1.CreateIngressResponse], error) {
	l := zap.S().With(
		"name", req.Msg.GetName(),
		"namespace", req.Msg.GetNamespace())
	l.Info("creating new ingress entry")
	app, err := s.getApplicationForIngress(ctx, req.Msg.GetName(), req.Msg.GetNamespace())
	if err != nil {
		l.Error("creating new ingress entry", err)
		return nil, err
	}
	return connect.NewResponse(&cwafv1.CreateIngressResponse{}),
		database.NewIngressFromRequest(req.Msg, app)
}

func (s *IngressService) getApplicationForIngress(
	ctx context.Context, name, namespace string) (
	*database.Application, error) {
	// if application already exists,
	// use the app id for ingress creation
	getAppResp, err := s.appSvc.GetApplication(
		ctx,
		connect.NewRequest(
			&cwafv1.GetApplicationRequest{
				NameOrId: &cwafv1.GetApplicationRequest_Name{
					Name: name,
				},
			},
		),
	)
	// all good return found application
	if err == nil {
		return &database.Application{ID: uint(getAppResp.Msg.Application.GetId())}, nil
	}
	// unexpected code, return error
	if connect.CodeOf(err) != connect.CodeNotFound {
		return nil, err
	}
	// application does not exist, create it
	createAppResp, err := s.appSvc.CreateApplication(ctx,
		connect.NewRequest(
			&cwafv1.CreateApplicationRequest{
				Name:        name,
				Namespace:   namespace,
				Description: "created automatically by discovery agent",
			}),
	)
	if err != nil {
		return &database.Application{ID: uint(createAppResp.Msg.GetId())}, err
	}
	return &database.Application{ID: uint(createAppResp.Msg.GetId())}, nil
}
