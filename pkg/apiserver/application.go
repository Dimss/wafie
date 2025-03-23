package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/database/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ApplicationService struct {
	cwafv1connect.UnimplementedApplicationServiceHandler
	db *gorm.DB
}

func NewApplicationService(db *gorm.DB) *ApplicationService {
	return &ApplicationService{
		db: db,
	}
}

func (s *ApplicationService) CreateApplication(
	ctx context.Context,
	req *connect.Request[cwafv1.CreateApplicationRequest]) (
	*connect.Response[cwafv1.CreateApplicationResponse], error) {
	zap.S().With(
		"name", req.Msg.GetName(),
		"namespace", req.Msg.GetNamespace()).
		Info("creating new application entry")
	app := model.NewApplicationFromRequest(req.Msg)
	dbRes := s.db.Create(&app)
	return connect.NewResponse(
		&cwafv1.CreateApplicationResponse{Id: uint32(app.ID)},
	), dbRes.Error
}

func (s *ApplicationService) GetApplication(
	ctx context.Context,
	req *connect.Request[cwafv1.GetApplicationRequest]) (
	*connect.Response[cwafv1.GetApplicationResponse], error) {

	app, err := model.GetApplicationByNameOrId(req.Msg, s.db)
	return connect.NewResponse(&cwafv1.GetApplicationResponse{
		Id:        uint32(app.ID),
		Name:      app.Name,
		Namespace: app.Namespace,
		Protected: app.Protected,
	}), err
}
