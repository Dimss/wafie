package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/models"
	"go.uber.org/zap"
)

type DataVersionService struct {
	cwafv1connect.UnimplementedDataVersionServiceHandler
	logger *zap.Logger
}

func NewDataVersionService(log *zap.Logger) *DataVersionService {
	return &DataVersionService{
		logger: log,
	}
}

func (s *DataVersionService) GetDataVersion(
	ctx context.Context,
	req *connect.Request[cwafv1.GetDataVersionRequest]) (
	*connect.Response[cwafv1.GetDataVersionResponse], error) {
	s.logger.Info("getting protection version")
	defer s.logger.Info("protection version retrieved")
	version, err := models.
		NewDataVersionModelSvc(nil, s.logger).
		GetVersionByTypeId(uint32(req.Msg.TypeId))
	if err != nil {
		s.logger.Error("error getting protection version", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(version.ToProto()), nil
}
