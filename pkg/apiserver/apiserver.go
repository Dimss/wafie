package apiserver

import (
	"connectrpc.com/grpcreflect"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
)

type ApiServer struct {
	logger *zap.Logger
}

func NewApiServer(log *zap.Logger) *ApiServer {

	return &ApiServer{logger: log}
}

func (s *ApiServer) Start() {
	s.logger.Info("starting API server")
	mux := http.NewServeMux()
	s.enableReflection(mux)
	s.registerHandlers(mux)
	go func() {
		http.ListenAndServe(":8080", h2c.NewHandler(mux, &http2.Server{}))
	}()
	s.logger.Info("server running on 0.0.0.0:8080")
}

func (s *ApiServer) registerHandlers(mux *http.ServeMux) {
	s.logger.Info("registering handlers")
	mux.Handle(
		cwafv1connect.NewApplicationServiceHandler(
			NewApplicationService(s.logger),
		),
	)
	mux.Handle(
		cwafv1connect.NewIngressServiceHandler(
			NewIngressService(s.logger),
		),
	)
	mux.Handle(
		cwafv1connect.NewProtectionServiceHandler(
			NewProtectionService(s.logger),
		),
	)
	mux.Handle(
		cwafv1connect.NewVirtualHostServiceHandler(
			NewVirtualHostService(s.logger),
		),
	)
	mux.Handle(
		cwafv1connect.NewDataVersionServiceHandler(
			NewDataVersionService(s.logger),
		),
	)

}

func (s *ApiServer) enableReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		cwafv1connect.IngressServiceName,
		cwafv1connect.AuthServiceName,
		cwafv1connect.ApplicationServiceName,
		cwafv1connect.ProtectionServiceName,
		cwafv1connect.VirtualHostServiceName,
		cwafv1connect.DataVersionServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
}
