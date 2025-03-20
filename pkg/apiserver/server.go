package apiserver

import (
	"connectrpc.com/grpcreflect"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/pkg/apiserver/ingress"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
)

type ApiServer struct{}

func NewApiServer() *ApiServer {
	return &ApiServer{}
}

func (s *ApiServer) Start() {
	mux := http.NewServeMux()
	reflector := grpcreflect.NewStaticReflector(
		cwafv1connect.IngressServiceName,
		cwafv1connect.AuthServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	path, handler := cwafv1connect.NewIngressServiceHandler(ingress.NewService())
	mux.Handle(path, handler)
	http.ListenAndServe(":8080", h2c.NewHandler(mux, &http2.Server{}))
}
