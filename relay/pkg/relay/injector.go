package relay

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
	"github.com/containernetworking/plugins/pkg/ns"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	CRISocketPath       = "unix:///run/containerd/containerd.sock" // Adjust for your specific runtime
	NetNsBasePath       = "/host/var/run/netns/"
	HealthCheckEndpoint = "http://127.0.0.1:8081"
)

type Injector struct {
	containerId     string
	nodeName        string
	pidNsRef        string // i.e /proc/2871780/ns/net
	netId           uint64 // net ns inode
	namedNetNs      string // i.e
	logger          *zap.Logger
	healthCheckAddr string
}

func NewInjector(containerId, nodeName string, logger *zap.Logger) (*Injector, error) {
	var err error
	i := &Injector{logger: logger, nodeName: nodeName, healthCheckAddr: HealthCheckEndpoint}
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
	logger.Debug(fmt.Sprintf("%+v", i))
	return i, nil
}

// Start idempotent method, will do nothing if instance already injected and running
// otherwise will clean up previous instance and start a new one
func (i *Injector) Start() error {
	if i.relayRunning() {
		i.logger.Info("relay running, no need to start",
			zap.String("containerId", i.containerId),
			zap.String("netns", i.namedNetNs))
		return nil
	}
	ctx, _ := context.WithCancel(context.Background())
	var netNs ns.NetNS
	defer func(netNs ns.NetNS) {
		if netNs != nil {
			netNs.Close()
		}
	}(netNs)

	netNs, err := ns.GetNS(i.namedNetNs)
	if err != nil {
		return err
	}
	return netNs.Do(func(_ ns.NetNS) error {
		i.logger.Info("network namespace set", zap.String("path", i.namedNetNs))
		cmd := exec.CommandContext(ctx,
			"wafie-relay",
			"start",
			"relay-instance",
		)
		return cmd.Start()
	})
}

func (i *Injector) namespacedHttpClient() *http.Client {
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
				nsFile, err := os.Open(i.pidNsRef)
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

func (i *Injector) relayRunning() (isRunning bool) {
	relayHealthCheck := healthv1connect.NewHealthClient(i.namespacedHttpClient(), i.healthCheckAddr)
	resp, err := relayHealthCheck.Check(context.Background(), connect.NewRequest(&healthv1.HealthCheckRequest{}))
	// if relay no running, expecting to get CodeUnavailable (ECONNREFUSED)
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return false
	}
	// TODO: need a way to monitor an errors and reactively fix them
	// in case of any error do nothing, i.e return true
	if err != nil {
		i.logger.Error("health check failed", zap.Error(err))
		return true
	}
	if resp.Msg.GetStatus() == healthv1.HealthCheckResponse_SERVING {
		return true
	}
	return false
}

func (i *Injector) setNamedNetNs() error {

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
		if stat.Ino == i.netId {
			i.namedNetNs = NetNsBasePath + entry.Name()
			return nil
		}
	}
	return nil
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
