package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	healthv1 "github.com/Dimss/wafie/api/gen/grpc/health/v1"
	"github.com/Dimss/wafie/api/gen/grpc/health/v1/healthv1connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"github.com/containernetworking/plugins/pkg/ns"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	ContainerdCRISock = "unix:///run/containerd/containerd.sock" // Adjust for your specific runtime
	CRIoCRISock       = "unix:///var/run/crio/crio.sock"
	InstanceApiAddr   = "http://127.0.0.1:8081"
)

type RelayInstanceSpec struct {
	containerId string
	runtimeSock string
	nodeName    string
	netnsPath   string
	logger      *zap.Logger
	apiAddr     string
	podName     string
}

func NewRelayInstanceSpec(containerId, podName, nodeName string, logger *zap.Logger) (*RelayInstanceSpec, error) {
	var err error
	i := &RelayInstanceSpec{
		logger:   logger.With(zap.String("podName", podName)),
		nodeName: nodeName,
		apiAddr:  InstanceApiAddr,
		podName:  podName,
	}
	// set container id
	if i.containerId, i.runtimeSock, err = parseContainerId(containerId); err != nil {
		return nil, err
	}
	if err := i.discoverNetnsPath(); err != nil {
		return nil, err
	}
	i.logger = logger.With(
		zap.String("containerId", containerId),
		zap.String("nodeName", nodeName),
		zap.String("podName", podName),
	)
	i.logger.Debug(fmt.Sprintf("%+v", i))
	return i, nil
}

func (s *RelayInstanceSpec) StopSpec() error {
	if !s.relayRunning() {
		return nil
	}
	_, err := wafiev1connect.NewRelayServiceClient(s.namespacedHttpClient(), s.apiAddr).
		StopRelay(context.Background(), connect.NewRequest(&wafiev1.StopRelayRequest{}))
	if err != nil {
		return err
	}
	return nil
}

func (s *RelayInstanceSpec) startRelay() error {
	if !s.relayRunning() {
		return nil
	}
	_, err := wafiev1connect.NewRelayServiceClient(s.namespacedHttpClient(), s.apiAddr).
		StartRelay(context.Background(), connect.NewRequest(&wafiev1.StartRelayRequest{}))
	if err != nil {
		return err
	}
	return nil
}

// StartSpec idempotent method, will do nothing if instance already injected and running
// otherwise will clean up previous instance and start a new one
func (s *RelayInstanceSpec) StartSpec() error {
	if !s.relayRunning() {
		if err := s.runRelayBinary(); err != nil {
			return err
		}
	}
	return s.startRelay()
}

func (s *RelayInstanceSpec) runRelayBinary() error {
	var netNs ns.NetNS
	defer func(netNs ns.NetNS) {
		if netNs != nil {
			netNs.Close()
		}
	}(netNs)
	netNs, err := ns.GetNS(s.netnsPath)
	if err != nil {
		return err
	}
	return netNs.Do(func(_ ns.NetNS) error {
		s.logger.Info("network namespace set", zap.String("path", s.netnsPath))
		cmd := exec.Command(
			"/usr/local/bin/wafie-relay",
			"start", "relay-instance",
		)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return cmd.Start()
	})
}

func (s *RelayInstanceSpec) namespacedHttpClient() *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				// Save current namespace (optional but safer)
				currentNS, err := os.Open("/proc/self/ns/net")
				if err != nil {
					return nil, err
				}
				defer currentNS.Close()
				// Switch to target namespace
				nsFile, err := os.Open(s.netnsPath)
				if err != nil {
					return nil, err
				}
				defer nsFile.Close()
				err = unix.Setns(int(nsFile.Fd()), unix.CLONE_NEWNET)
				if err != nil {
					return nil, err
				}
				// Restore original namespace when done
				defer unix.Setns(int(currentNS.Fd()), unix.CLONE_NEWNET)
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
}

func (s *RelayInstanceSpec) relayRunning() (isRunning bool) {
	relayHealthCheck := healthv1connect.NewHealthClient(s.namespacedHttpClient(), s.apiAddr)
	resp, err := relayHealthCheck.Check(context.Background(), connect.NewRequest(&healthv1.HealthCheckRequest{}))
	// if relay no running, expecting to get CodeUnavailable (ECONNREFUSED)
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return false
	}
	// TODO: need a way to monitor an errors and reactively fix them
	// in case of any error do nothing, s.e return true
	if err != nil {
		s.logger.Error("health check failed", zap.Error(err))
		return true
	}
	if resp.Msg.GetStatus() == healthv1.HealthCheckResponse_SERVING {
		return true
	}
	return false
}

func (s *RelayInstanceSpec) discoverNetnsPath() error {
	conn, err := grpc.NewClient(
		s.runtimeSock,
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
		ContainerId: s.containerId,
		Verbose:     true,
	}
	response, err := client.ContainerStatus(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get container status: %v\n", err)
	}
	if s.netnsPath, err = getContainerNetworkNs(response); err != nil {
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
		fmt.Printf("failed to unmarshal namespaces: %v\n", err)
	}
	for _, ns := range namespaces {
		if ns.Key == "network" {
			return ns.Path, nil
		}
	}
	return "", fmt.Errorf("failed to find network namespace")
}

func parseContainerId(containerId string) (id string, runtimeSock string, err error) {
	slice := strings.Split(containerId, "://")
	if len(slice) != 2 {
		return "", "", fmt.Errorf("unable to parse container id")
	}
	if slice[0] == "cri-o" {
		return slice[1], CRIoCRISock, nil
	}
	if slice[0] == "containerd" {
		return slice[1], ContainerdCRISock, nil
	}
	return "", "", fmt.Errorf("unable to detect container runtime")

}
