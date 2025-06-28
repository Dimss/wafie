package apiserver

import (
	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	v1 "github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
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
	compress1KB := connect.WithCompressMinBytes(1024)
	mux.Handle(
		grpchealth.NewHandler(
			NewHealthCheckService(s.logger),
			compress1KB,
		),
	)
	mux.Handle(
		v1.NewApplicationServiceHandler(
			NewApplicationService(s.logger),
			compress1KB,
		),
	)
	mux.Handle(
		v1.NewIngressServiceHandler(
			NewIngressService(s.logger),
			compress1KB,
		),
	)
	mux.Handle(
		v1.NewProtectionServiceHandler(
			NewProtectionService(s.logger),
			compress1KB,
		),
	)
	mux.Handle(
		v1.NewVirtualHostServiceHandler(
			NewVirtualHostService(s.logger),
			compress1KB,
		),
	)
	mux.Handle(
		v1.NewDataVersionServiceHandler(
			NewDataVersionService(s.logger),
			compress1KB,
		),
	)
}

func (s *ApiServer) enableReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		v1.IngressServiceName,
		v1.AuthServiceName,
		v1.ApplicationServiceName,
		v1.ProtectionServiceName,
		v1.VirtualHostServiceName,
		v1.DataVersionServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
}
