package main

import (
	"fmt"
	"github.com/Dimss/wafie/xproc/pkg/processor"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(s, &processor.ExternalProcessor{})

	fmt.Println("External Processor server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
