package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	CRISocketPath = "unix:///run/containerd/containerd.sock" // Adjust for your specific runtime
)

type Injector struct {
	containerId string
	nodeName    string
	pidNsRef    string // i.e /proc/2871780/ns/net
	netId       uint64 // net ns inode
	namedNetNs  string // i.e
	logger      *zap.Logger
}

func NewInjector(containerId, nodeName string, logger *zap.Logger) (*Injector, error) {
	var err error
	i := &Injector{logger: logger}
	// set container id
	if i.containerId, err = parseContainerId(containerId); err != nil {
		return nil, err
	}
	// set ns ref, i.e: /proc/2871780/ns/net
	if err = i.setPidNsRef(); err != nil {
		return nil, err
	}
	if err = i.pidNsRefToNetId(); err != nil {
		return nil, err
	}
	logger.Debug(fmt.Sprintf("%+v", i))
	return i, nil
}

func (i *Injector) pidNsRefToNetId() error {
	entry, err := os.Readlink(i.pidNsRef)
	if err != nil {
		return err
	}
	r, err := regexp.Compile("\\d")
	if err != nil {
		return err
	}
	res := strings.Join(r.FindAllString(entry, -1), "")
	i.netId, err = strconv.ParseUint(res, 10, 64)
	return err
}

func (i *Injector) setPidNsRef() error {
	conn, err := grpc.NewClient(
		CRISocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %v", err)
	}
	defer conn.Close()
	client := runtimeapi.NewRuntimeServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request := &runtimeapi.ContainerStatusRequest{
		ContainerId: i.containerId,
		Verbose:     true,
	}
	response, err := client.ContainerStatus(ctx, request)
	if err != nil {
		return fmt.Errorf("Failed to get container status: %v\n", err)
	}
	if i.pidNsRef, err = getContainerNetworkNs(response); err != nil {
		return err
	}
	return nil
}

func getContainerNetworkNs(containerStatusResponse *runtimeapi.ContainerStatusResponse) (string, error) {
	infoMap := make(map[string]interface{})
	if _, ok := containerStatusResponse.Info["info"]; !ok {
		return "", fmt.Errorf("info not found in response")
	}
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

func parseContainerId(containerId string) (string, error) {
	slice := strings.Split(containerId, "://")
	if len(slice) != 2 {
		return "", fmt.Errorf("unable to parse container id")
	}
	return slice[1], nil
}
