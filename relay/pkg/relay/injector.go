package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	CRISocketPath = "unix:///run/containerd/containerd.sock" // Adjust for your specific runtime

)

type Injector struct {
	ContainerId string
	NodeName    string
}

func NewInjector(containerId, nodeName string) *Injector {
	return &Injector{
		containerId,
		nodeName,
	}
}

func (i *Injector) GetPids() {

	conn, err := grpc.NewClient(
		CRISocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		//	return net.Dial("unix", addr)
		//}),
	)
	if err != nil {
		fmt.Errorf("failed to create gRPC client: %v", err)
	}
	defer conn.Close()
	client := runtimeapi.NewRuntimeServiceClient(conn)

	// 3. Call ContainerStatus
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := &runtimeapi.ContainerStatusRequest{
		ContainerId: "4ceea125911d135398f8d4a4c4e60953ca32c048f435f6efb24ca3040ea1fb2a",
		Verbose:     true, // Request verbose output for more details
	}

	response, err := client.ContainerStatus(ctx, request)
	if err != nil {
		fmt.Printf("Failed to get container status: %v\n", err)
		return
	}
	nsName, err := getContainerNetworkNs(response)
	if err != nil {
		fmt.Printf("Failed to get container network namespace: %v\n", err)
	}
	fmt.Printf("Container network namespace: %v\n", nsName)

}

func getContainerNetworkNs(containerStatusResponse *runtimeapi.ContainerStatusResponse) (string, error) {
	infoMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(containerStatusResponse.Info["info"]), &infoMap); err != nil {
		fmt.Printf("Failed to unmarshal info: %v\n", err)
	}
	nsUnstructured := infoMap["runtimeSpec"].(map[string]interface{})["linux"].(map[string]interface{})["namespaces"].([]interface{})
	res, err := json.Marshal(nsUnstructured)
	if err != nil {
		return "", err
	}
	type namespace struct {
		Key  string `json:"type"`
		Path string `json:"path,omitempty"`
	}
	var namespaces []namespace
	err = json.Unmarshal(res, &namespaces)
	if err != nil {
		fmt.Printf("Failed to unmarshal namespaces: %v\n", err)
	}
	for _, ns := range namespaces {
		if ns.Key == "network" {
			return ns.Path, nil
		}
	}
	return "", fmt.Errorf("failed to find network namespace")
}
