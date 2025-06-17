package main

/*
#cgo LDFLAGS: -lkubeguard
#include <stdlib.h>
#include <kubeguard/kubeguardlib.h>
*/
import "C"
import (
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	C.kg_library_init(C.CString("/config"))
	c := config{}
	http.RegisterHttpFilterFactoryAndConfigParser("kubeguard", kubeGuardFilterFactory, c)

}

type config struct {
}

func (c config) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return nil, nil
}

func (c config) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	return nil
}

func kubeGuardFilterFactory(config interface{}, callbacks api.FilterCallbackHandler) api.StreamFilter {
	return &filter{
		callbacks: callbacks,
		logger:    applogger.NewLogger(),
	}
}

func main() {
	// KubeGuard ModSecurity Envoy HTTP filter
	// compiled as a shared object (.so) for use with Envoy.
	// depends on the kubeguard (kubeguard.so) library and kubeguard/kubeguardlib.h files
	// to build: go build -ldflags='-s -w' -o ./kubeguard-modsec.so -buildmode=c-shared ./cmd/modsecfilter
}
