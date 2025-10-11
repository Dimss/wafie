package apisrv

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	wafiev1connect.UnimplementedRelayServiceHandler
	logger     *zap.Logger
	listenAddr string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		logger:     applogger.NewLogger(),
		listenAddr: listenAddr,
	}
}

func (s *Server) Serve() {
	go func() {
		s.logger.Info("starting health check server", zap.String("address", s.listenAddr))
		mux := http.NewServeMux()
		mux.Handle(grpchealth.NewHandler(s))
		mux.Handle(wafiev1connect.NewRelayServiceHandler(s))
		go func() {
			if err := http.ListenAndServe(s.listenAddr, h2c.NewHandler(mux, &http2.Server{})); err != nil {
				s.logger.Error("failed to start health check server", zap.Error(err))
			}
		}()
	}()
}

func (s *Server) Check(context.Context, *grpchealth.CheckRequest) (*grpchealth.CheckResponse, error) {
	s.logger.Debug("health check check request received")
	return &grpchealth.CheckResponse{Status: grpchealth.StatusServing}, nil
}

func (s *Server) StopRelay(
	ctx context.Context,
	req *connect.Request[v1.StopRelayRequest]) (
	*connect.Response[v1.StopRelayResponse], error) {
	s.logger.Debug("terminating relay instance")
	return connect.NewResponse(&v1.StopRelayResponse{}), nil
}
