package processor

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"io"
	"log"
)

type ExternalProcessor struct {
	extproc.UnimplementedExternalProcessorServer
}

func (s *ExternalProcessor) Process(stream extproc.ExternalProcessor_ProcessServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var resp *extproc.ProcessingResponse
		switch req.Request.(type) {
		case *extproc.ProcessingRequest_RequestHeaders:
			log.Println("Processing request headers")
			resp = &extproc.ProcessingResponse{
				Response: &extproc.ProcessingResponse_RequestHeaders{
					RequestHeaders: &extproc.HeadersResponse{
						Response: &extproc.CommonResponse{
							//Status: extproc.CommonResponse_CONTINUE,
							HeaderMutation: &extproc.HeaderMutation{
								SetHeaders: []*core.HeaderValueOption{
									{
										Header: &core.HeaderValue{
											Key:      "host",
											RawValue: []byte("wp.10.100.102.92.nip.io"),
										},
									},
									{
										Header: &core.HeaderValue{
											Key:      "foo",
											RawValue: []byte("bar-bar-bar"),
										},
									},
								},
							},
						},
					},
				},
			}

		case *extproc.ProcessingRequest_ResponseHeaders:
			log.Println("Processing response headers")
			resp = &extproc.ProcessingResponse{
				Response: &extproc.ProcessingResponse_ResponseHeaders{
					ResponseHeaders: &extproc.HeadersResponse{},
				},
			}

		case *extproc.ProcessingRequest_RequestBody:
			log.Println("Processing request body")
			resp = &extproc.ProcessingResponse{
				Response: &extproc.ProcessingResponse_RequestBody{
					RequestBody: &extproc.BodyResponse{},
				},
			}

		case *extproc.ProcessingRequest_ResponseBody:
			log.Println("Processing response body")
			resp = &extproc.ProcessingResponse{
				Response: &extproc.ProcessingResponse_ResponseBody{
					ResponseBody: &extproc.BodyResponse{},
				},
			}
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
