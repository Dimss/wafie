package router

import (
	"connectrpc.com/connect"
	"context"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Service struct {
	appSvcClient cwafv1connect.ApplicationServiceClient
}

func NewRouteService(apiAddr string) *Service {
	return &Service{
		appSvcClient: cwafv1connect.NewApplicationServiceClient(
			http.DefaultClient, apiAddr),
	}
}

func (s *Service) Start() {

	go func() {
		for {
			zap.S().Info("fetching protected ingresses from api... ")
			if err := s.getProtectedIngresses(); err != nil {
				zap.S().Warn("failed to fetch protected ingresses", zap.Error(err))
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func (s *Service) getProtectedIngresses() error {
	req := &cwafv1.ListApplicationsRequest{
		Options: &cwafv1.ListApplicationsOptions{
			Protected: cwafv1.AppProtected_YES,
		},
	}
	apps, err := s.appSvcClient.ListApplications(
		context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	zap.S().Infof("got %d apps for protection", len(apps.Msg.Applications))
	return nil

}
