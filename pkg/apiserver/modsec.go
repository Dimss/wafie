package apiserver

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
)

type ModSecProtectionService struct {
	cwafv1connect.UnimplementedProtectionServiceHandler
	logger *zap.Logger
}

func NewModSecProtectionService(log *zap.Logger) *ModSecProtectionService {
	return &ModSecProtectionService{
		logger: log,
	}
}

func (s *ModSecProtectionService) CreateProtection(
	ctx context.Context, req *connect.Request[cwafv1.CreateProtectionRequest]) (
	*connect.Response[cwafv1.CreateProtectionResponse], error) {
	
	return nil, nil
}
