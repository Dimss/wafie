package main

/*
#cgo LDFLAGS: -lkubeguard
#include <stdlib.h>
#include <kubeguard/kubeguardlib.h>
*/
import "C"
import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	path := "/example.conf"
	rulesPath := C.CString(path)
	C.kg_library_init(rulesPath)
	c := config{}
	http.RegisterHttpFilterFactoryAndConfigParser("kubeguard", myFactory, c)

}

type config struct {
}

func (c config) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	return nil, nil
}

func (c config) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	return nil
}

func myFactory(config interface{}, callbacks api.FilterCallbackHandler) api.StreamFilter {
	return &filter{
		callbacks: callbacks,
	}
}

func main() {

}
