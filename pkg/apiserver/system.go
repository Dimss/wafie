package apiserver

import (
	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"context"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/models"
	"go.uber.org/zap"
)

type SystemService struct {
	cwafv1connect.UnimplementedSystemServiceHandler
	logger *zap.Logger
}

func NewSystemService(log *zap.Logger) *SystemService {
	return &SystemService{
		logger: log,
	}
}

func (s *SystemService) Check(context.Context, *grpchealth.CheckRequest) (*grpchealth.CheckResponse, error) {
	systemModelSvc := models.NewSystemModelSvc(nil, s.logger)
	if err := systemModelSvc.Ping(); err != nil {
		s.logger.Error("database ping failed", zap.Error(err))
		return &grpchealth.CheckResponse{Status: grpchealth.StatusNotServing}, connect.NewError(connect.CodeInternal, err)
	}
	return &grpchealth.CheckResponse{Status: grpchealth.StatusServing}, nil
}
