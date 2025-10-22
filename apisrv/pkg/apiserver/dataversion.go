package apiserver

import (
	"context"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/apisrv/internal/models"
	"go.uber.org/zap"
)

type DataVersionService struct {
	wafiev1connect.UnimplementedDataVersionServiceHandler
	logger *zap.Logger
}

func NewDataVersionService(log *zap.Logger) *DataVersionService {
	return &DataVersionService{
		logger: log,
	}
}

func (s *DataVersionService) GetDataVersion(
	ctx context.Context,
	req *connect.Request[wafiev1.GetDataVersionRequest]) (
	*connect.Response[wafiev1.GetDataVersionResponse], error) {
	version, err := models.
		NewDataVersionModelSvc(nil, s.logger).
		GetVersionByTypeId(uint32(req.Msg.TypeId))
	if err != nil {
		s.logger.Error("error getting protection version", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(version.ToProto()), nil
}
