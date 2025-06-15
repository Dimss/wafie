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
	"strings"
	"unsafe"
)

type filter struct {
	callbacks   api.FilterCallbackHandler
	evalRequest C.EvaluationRequest
	//conf      configuration
}

func (f *filter) evaluationRequestHeaders(allHeaders map[string][]string) *C.EvaluationRequestHeader {
	var i int
	var headers = (*C.EvaluationRequestHeader)(
		C.malloc(
			C.size_t(unsafe.Sizeof(C.EvaluationRequestHeader{})) * C.size_t(len(allHeaders)),
		),
	)
	for key, value := range allHeaders {
		hdr := (*C.EvaluationRequestHeader)(
			unsafe.Pointer(uintptr(unsafe.Pointer(headers)) + uintptr(i)*
				unsafe.Sizeof(C.EvaluationRequestHeader{})))
		hdr.key = (*C.uchar)(unsafe.Pointer(C.CString(key)))
		hdr.value = (*C.uchar)(unsafe.Pointer(C.CString(strings.Join(value, ""))))
		i++
	}
	return headers
}

func (f *filter) evaluationRequest(headerMap api.RequestHeaderMap) {
	var clientIp, httpVersion string
	clientIp, _ = headerMap.Get("X-Forwarded-For")
	httpVersion, _ = f.callbacks.StreamInfo().Protocol()
	f.evalRequest.client_ip = C.CString(clientIp)
	f.evalRequest.uri = C.CString(headerMap.Host() + headerMap.Path())
	f.evalRequest.http_method = C.CString(headerMap.Method())
	f.evalRequest.http_version = C.CString(httpVersion)
	f.evalRequest.headers_count = C.size_t(len(headerMap.GetAllHeaders()))
	f.evalRequest.headers = f.evaluationRequestHeaders(headerMap.GetAllHeaders())
	f.evalRequest.body = nil
	C.kg_init_request_transaction(&f.evalRequest)
}

func (f *filter) freeEvaluationRequest() {
	C.free(unsafe.Pointer(f.evalRequest.client_ip))
	C.free(unsafe.Pointer(f.evalRequest.uri))
	C.free(unsafe.Pointer(f.evalRequest.http_method))
	C.free(unsafe.Pointer(f.evalRequest.http_version))
	for i := 0; i < int(f.evalRequest.headers_count); i++ {
		hdr := (*C.EvaluationRequestHeader)(
			unsafe.Pointer(uintptr(unsafe.Pointer(f.evalRequest.headers)) + uintptr(i)*
				unsafe.Sizeof(C.EvaluationRequestHeader{})))
		C.free(unsafe.Pointer(hdr.key))
		C.free(unsafe.Pointer(hdr.value))
	}
	C.free(unsafe.Pointer(f.evalRequest.headers))
}

func (f *filter) DecodeHeaders(headerMap api.RequestHeaderMap, b bool) api.StatusType {
	fmt.Println(">>>>>>>>>>>>>>>>>> RUNNING CWAF HTTP FILTER <<<<<<<<<<<<<<<<<<<")
	f.evaluationRequest(headerMap)
	//C.kg_add_rule(C.CString("SecRule REMOTE_ADDR \"@ipMatch 10.244.0.31\" \"id:203948180384," +
	//	"phase:0,deny,status:403,msg:'Blocking connection from specific IP'\""))
	if C.kg_process_request_headers(&f.evalRequest) != 0 {
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(403,
			"Opa opa, access denied!!!", nil, 0, "some details here")
		return api.LocalReply
	}

	defer fmt.Println(">>>>>>>>>>>>>>>>>> DONE <<<<<<<<<<<<<<<<<<<")

	return api.Continue
}

func (f *filter) DecodeData(instance api.BufferInstance, b bool) api.StatusType {

	body := instance.Bytes()
	fmt.Println(">>>>>>>>>>>>>. DECODE DATA >>>>>>>>>>>>>")
	defer fmt.Println(">>>>>>>>>>>>>. DECODE DATA DONE >>>>>>>>>>>>>")
	fmt.Println(string(body))
	return api.Continue
}

func (f *filter) DecodeTrailers(trailerMap api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeHeaders(headerMap api.ResponseHeaderMap, b bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeData(instance api.BufferInstance, b bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeTrailers(trailerMap api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) OnLog(
	headerMap api.RequestHeaderMap,
	trailerMap api.RequestTrailerMap,
	headerMap2 api.ResponseHeaderMap,
	trailerMap2 api.ResponseTrailerMap) {
}

func (f *filter) OnLogDownstreamStart(headerMap api.RequestHeaderMap) {
}

func (f *filter) OnLogDownstreamPeriodic(
	headerMap api.RequestHeaderMap,
	trailerMap api.RequestTrailerMap,
	headerMap2 api.ResponseHeaderMap,
	trailerMap2 api.ResponseTrailerMap) {
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
	defer fmt.Println(fmt.Sprintf("distrotying filter instance, reason: %d\n", reason))
	f.freeEvaluationRequest()

}

func (f *filter) OnStreamComplete() {

}
