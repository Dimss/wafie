package healthchecksrv

import (
	"context"
	"net/http"

	"connectrpc.com/grpchealth"
	"github.com/Dimss/wafie/internal/applogger"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
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
