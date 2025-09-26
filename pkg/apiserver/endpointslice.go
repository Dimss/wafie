package apiserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EndpointSliceService struct {
	v1.UnimplementedEndpointSliceServiceHandler
	logger *zap.Logger
}

func NewEndpointSliceService(logger *zap.Logger) *EndpointSliceService {
	return &EndpointSliceService{
		logger: logger,
	}
}

func (s *EndpointSliceService) CreateEndpointSlice(ctx context.Context,
	req *connect.Request[cwafv1.CreateEndpointSliceRequest]) (
	*connect.Response[cwafv1.CreateEndpointSliceResponse], error) {
	s.logger.With(
		zap.String("name", req.Msg.EndpointSlice.UpstreamHost)).
		Info("creating new endpoint slice entry")
	defer s.logger.Info("endpoint slice entry created")
	epSliceModelSvc := models.NewEndpointSliceModelSvc(nil, s.logger)
	eps, err := epSliceModelSvc.NewEndpointSliceFromRequest(req.Msg)
	if err != nil {
		// if no upstream exists, meaning we do not
		// care about this endpoint slice
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			s.logger.Info("no upstream exists for given endpoint slice")
			return connect.NewResponse(&cwafv1.CreateEndpointSliceResponse{}), nil
		}
		return connect.NewResponse(&cwafv1.CreateEndpointSliceResponse{}),
			connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(
		&cwafv1.CreateEndpointSliceResponse{
			EndpointSliceId: uint32(eps.ID),
			EndpointSlice:   eps.ToProto(),
		},
	)
	return resp, nil
}
