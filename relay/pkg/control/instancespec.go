package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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
	CRISocketPath   = "unix:///run/containerd/containerd.sock" // Adjust for your specific runtime
	NetNsBasePath   = "/host/var/run/netns/"
	InstanceApiAddr = "http://127.0.0.1:8081"
)

type RelayInstanceSpec struct {
	containerId string
	nodeName    string
	pidNsRef    string // i.e /proc/2871780/ns/net
	netId       uint64 // net ns inode
	namedNetNs  string // i.e /host/var/run/netns/cni-52d62ed1-ad22-d5fa-332e-21b9d0e82250
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
	if err = i.setNamedNetNs(); err != nil {
		return nil, err
	}
	i.logger = logger.With(
		zap.String("containerId", containerId),
		zap.String("nodeName", nodeName),
		zap.String("podName", podName),
		zap.String("netns", i.namedNetNs),
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
	netNs, err := ns.GetNS(s.namedNetNs)
	if err != nil {
		return err
	}
	return netNs.Do(func(_ ns.NetNS) error {
		s.logger.Info("network namespace set", zap.String("path", s.namedNetNs))
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
				nsFile, err := os.Open(s.pidNsRef)
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

func (s *RelayInstanceSpec) setNamedNetNs() error {

	netNsEntries, err := os.ReadDir(NetNsBasePath)
	if err != nil {
		return err
	}
	for _, entry := range netNsEntries {
		if entry.IsDir() {
			continue
		}
		fileInfo, err := os.Stat(NetNsBasePath + entry.Name())
		if err != nil {
			return err
		}
		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("could not get syscall.Stat_t from FileInfo")
		}
		// find the NetNs mount path base on the net id (inode)
		if stat.Ino == s.netId {
			s.namedNetNs = NetNsBasePath + entry.Name()
			return nil
		}
	}
	return nil
}

func (s *RelayInstanceSpec) pidNsRefToNetId() error {
	entry, err := os.Readlink(s.pidNsRef)
	if err != nil {
		return err
	}
	r, err := regexp.Compile("\\d")
	if err != nil {
		return err
	}
	res := strings.Join(r.FindAllString(entry, -1), "")
	s.netId, err = strconv.ParseUint(res, 10, 64)
	return err
}

func (s *RelayInstanceSpec) setPidNsRef() error {
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
		ContainerId: s.containerId,
		Verbose:     true,
	}
	response, err := client.ContainerStatus(ctx, request)
	if err != nil {
		return fmt.Errorf("Failed to get container status: %v\n", err)
	}
	if s.pidNsRef, err = getContainerNetworkNs(response); err != nil {
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

func parseContainerId(containerId string) (string, error) {
	slice := strings.Split(containerId, "://")
	if len(slice) != 2 {
		return "", fmt.Errorf("unable to parse container id")
	}
	return slice[1], nil
}
