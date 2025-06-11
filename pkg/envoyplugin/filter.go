package main

/*
#cgo LDFLAGS: -lkubeguard
#include <stdlib.h>
#include <kubeguard/kubeguardlib.h>
*/
import "C"
import (
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"unsafe"
)

type filter struct {
	callbacks api.FilterCallbackHandler
	//conf      configuration
}

func (f filter) DecodeHeaders(headerMap api.RequestHeaderMap, b bool) api.StatusType {
	fmt.Println(">>>>>>>>>>>>>>>>>> RUNNING CWAF HTTP FILTER <<<<<<<<<<<<<<<<<<<")
	path := "/example.conf"
	rulesPath := C.CString(path)
	defer C.free(unsafe.Pointer(rulesPath))
	C.dump_rules(rulesPath)
	defer fmt.Println(">>>>>>>>>>>>>>>>>> DONE <<<<<<<<<<<<<<<<<<<")
	return api.Continue
}

func (f filter) DecodeData(instance api.BufferInstance, b bool) api.StatusType {
	return api.Continue
}

func (f filter) DecodeTrailers(trailerMap api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f filter) EncodeHeaders(headerMap api.ResponseHeaderMap, b bool) api.StatusType {
	return api.Continue
}

func (f filter) EncodeData(instance api.BufferInstance, b bool) api.StatusType {
	return api.Continue
}

func (f filter) EncodeTrailers(trailerMap api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

func (f filter) OnLog(
	headerMap api.RequestHeaderMap,
	trailerMap api.RequestTrailerMap,
	headerMap2 api.ResponseHeaderMap,
	trailerMap2 api.ResponseTrailerMap) {
}

func (f filter) OnLogDownstreamStart(headerMap api.RequestHeaderMap) {
}

func (f filter) OnLogDownstreamPeriodic(
	headerMap api.RequestHeaderMap,
	trailerMap api.RequestTrailerMap,
	headerMap2 api.ResponseHeaderMap,
	trailerMap2 api.ResponseTrailerMap) {
}

func (f filter) OnDestroy(reason api.DestroyReason) {}

func (f filter) OnStreamComplete() {}
