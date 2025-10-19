package apiserver

import (
	"context"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/apisrv/internal/models"
	"go.uber.org/zap"
)

type IngressService struct {
	wafiev1connect.UnimplementedIngressServiceHandler
	appSvc *ApplicationService
	logger *zap.Logger
}

func NewIngressService(log *zap.Logger) *IngressService {
	return &IngressService{
		logger: log,
		appSvc: NewApplicationService(log),
	}
}

func (s *IngressService) CreateIngress(
	ctx context.Context,
	req *connect.Request[wafiev1.CreateIngressRequest]) (
	*connect.Response[wafiev1.CreateIngressResponse], error) {
	//setup logger with request context
	l := s.logger.With(zap.String("name", req.Msg.Ingress.Name))
	l.Info("creating new ingress entry")
	//app, err := s.getApplicationForIngress(ctx, req.Msg.Ingress.Name)
	//if err != nil {
	//	l.Error("creating new ingress entry", zap.Error(err))
	//	return nil, err
	//}
	ingressModelSvc := models.NewIngressModelSvc(nil, l)
	return connect.NewResponse(&wafiev1.CreateIngressResponse{}),
		ingressModelSvc.NewIngressFromRequest(req.Msg)
}

func (s *IngressService) getApplicationForIngress(ctx context.Context, name string) (
	*models.Application, error) {
	// if application already exists,
	// use the app id for ingress creation
	getAppResp, err := s.appSvc.GetApplication(
		ctx,
		connect.NewRequest(
			&wafiev1.GetApplicationRequest{ // ToDo: implement
				//NameOrId: &wafiev1.GetApplicationRequest_Name{
				//	Name: name,
				//},
			},
		),
	)
	// all good return found application
	if err == nil {
		return &models.Application{ID: uint(getAppResp.Msg.Application.GetId())}, nil
	}
	// unexpected code, return error
	if connect.CodeOf(err) != connect.CodeNotFound {
		return nil, err
	}
	// application does not exist, create it
	createAppResp, err := s.appSvc.CreateApplication(ctx,
		connect.NewRequest(
			&wafiev1.CreateApplicationRequest{
				Name: name,
			}),
	)
	if err != nil {
		return &models.Application{ID: uint(createAppResp.Msg.GetId())}, err
	}
	return &models.Application{ID: uint(createAppResp.Msg.GetId())}, nil
}
