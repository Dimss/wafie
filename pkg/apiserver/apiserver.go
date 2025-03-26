package apiserver

import (
	"connectrpc.com/grpcreflect"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
	"net/http"
)

type ApiServer struct {
	db *gorm.DB
}

func NewApiServer(db *gorm.DB) *ApiServer {

	return &ApiServer{db: db}
}

func (s *ApiServer) Start() {
	zap.S().Info("starting API server")
	mux := http.NewServeMux()
	s.enableReflection(mux)
	s.registerHandlers(mux)
	go func() {
		http.ListenAndServe(":8080", h2c.NewHandler(mux, &http2.Server{}))
	}()
	zap.S().Info("server running on 0.0.0.0:8080")
}

func (s *ApiServer) registerHandlers(mux *http.ServeMux) {
	mux.Handle(
		cwafv1connect.NewIngressServiceHandler(
			NewIngressService(),
		),
	)
	mux.Handle(
		cwafv1connect.NewApplicationServiceHandler(
			NewApplicationService(),
		),
	)
}

func (s *ApiServer) enableReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		cwafv1connect.IngressServiceName,
		cwafv1connect.AuthServiceName,
		cwafv1connect.ApplicationServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
}
