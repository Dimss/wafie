package controller

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"os/exec"

	//"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	cycleTime     = 1 * time.Second
	nginxBinPatch = "/opt/app/nginx/sbin/nginx"
)

type CycleStats struct {
	Created uint32
	Deleted uint32
}

type ConfigState struct {
	ConfigId int
	Checksum string
	Spec     []byte
	BasePath string
}

type Nginx struct {
	VirtualHostsConfigPath string
	logger                 *zap.Logger
	VirtualHostSvcClient   cwafv1connect.VirtualHostServiceClient
	VirtualHostsState      map[string]uint32
	ActualState            []*ConfigState
	DesiredState           []*ConfigState
	CycleStats             *CycleStats
}

func newStateFromVirtualHostFile(basePath, configFileName string) (*ConfigState, error) {
	parsedState := strings.Split(
		strings.ReplaceAll(configFileName, ".conf", ""), "-",
	)
	if len(parsedState) != 2 {
		fmt.Println("error parsing config file name")
		return nil, nil
	}
	parsedId, err := strconv.ParseInt(parsedState[0], 10, 32)
	if err != nil {
		return nil, err
	}
	cfg := &ConfigState{}
	cfg.ConfigId = int(parsedId)
	cfg.Checksum = parsedState[1]
	cfg.BasePath = basePath
	return cfg, nil
}

func newStateFromVirtualHostApi(basePath string, vh *cwafv1.VirtualHost) *ConfigState {
	return &ConfigState{
		ConfigId: int(vh.Id),
		Checksum: vh.Checksum,
		Spec:     []byte(vh.Spec),
		BasePath: basePath,
	}
}

func NewNginxController(
	configPath string,
	logger *zap.Logger,
	virtualHostSvcClient cwafv1connect.VirtualHostServiceClient) *Nginx {

	return &Nginx{
		VirtualHostsConfigPath: configPath,
		logger:                 logger,
		VirtualHostSvcClient:   virtualHostSvcClient,
		CycleStats:             &CycleStats{},
	}
}

func (n *Nginx) StartCycleLoop() {
	// start nginx
	if err := n.start(); err != nil {
		n.logger.Error("error starting nginx", zap.Error(err))
		return
	}
	// start reconcile loop
	for {
		time.Sleep(cycleTime)
		// reset cycle stats
		n.CycleStats.reset()
		// set actual state
		if err := n.actualState(); err != nil {
			n.logger.Error("error discovering actual state", zap.Error(err))
			continue
		}
		// set desired state
		if err := n.desiredState(); err != nil {
			n.logger.Error("error discovering desired state", zap.Error(err))
			continue
		}
		// apply desired state
		if err := n.apply(); err != nil {
			n.logger.Error("error applying desired state", zap.Error(err))
			continue
		}
		// reload nginx if needed
		if n.shouldReload() {
			l := n.logger.
				With(zap.Uint32("created", n.CycleStats.Created)).
				With(zap.Uint32("deleted", n.CycleStats.Deleted))
			if err := n.reload(); err != nil {
				l.Error("error reloading nginx", zap.Error(err))
				continue
			} else {
				l.Info("new config state applied successfully")
			}
		} else {
			n.logger.Info("no state changes detected, no reload needed")
		}
	}
}

func (n *Nginx) actualState() error {
	// reset current state
	n.resetActualState()
	// discover actual state
	virtualHostsFiles, err := os.ReadDir(n.VirtualHostsConfigPath)
	if err != nil {
		return err
	}
	//reset current state
	n.ActualState = make([]*ConfigState, len(virtualHostsFiles))
	for idx, vhFile := range virtualHostsFiles {
		vh, err := newStateFromVirtualHostFile(n.VirtualHostsConfigPath, vhFile.Name())
		if err != nil {
			n.logger.Error("error parsing virtual host config file", zap.Error(err))
			return err
		}
		n.ActualState[idx] = vh
	}
	return nil
}

func (n *Nginx) desiredState() error {
	// reset desired state
	n.resetDesiredState()
	// discover desired state
	virtualHosts, err := n.VirtualHostSvcClient.ListVirtualHosts(context.Background(),
		connect.NewRequest(&cwafv1.ListVirtualHostsRequest{}))
	if err != nil {
		return err
	}
	n.DesiredState = make([]*ConfigState, len(virtualHosts.Msg.VirtualHosts))
	for idx, vs := range virtualHosts.Msg.VirtualHosts {
		n.DesiredState[idx] = newStateFromVirtualHostApi(n.VirtualHostsConfigPath, vs)
	}
	return nil
}

func (n *Nginx) resetActualState() {
	n.ActualState = nil
}

func (n *Nginx) resetDesiredState() {
	n.DesiredState = nil
}

func (n *Nginx) apply() error {
	if err := n.removeVirtualServer(); err != nil {
		return err
	}
	if err := n.addVirtualServer(); err != nil {
		return err
	}
	return nil
}

// removeVirtualServer: removes not existing virtual hosts files
func (n *Nginx) removeVirtualServer() error {
	for _, actualVs := range n.ActualState {
		if !actualVs.exists(n.DesiredState) {
			if err := actualVs.remove(); err != nil {
				return err
			} else {
				n.CycleStats.Deleted++
			}
		}
	}
	return nil
}

// addVirtualServer: addVirtualServer new virtual hosts files
func (n *Nginx) addVirtualServer() error {
	for _, desiredVs := range n.DesiredState {
		if !desiredVs.exists(n.ActualState) {
			if err := desiredVs.create(); err != nil {
				return err
			} else {
				n.CycleStats.Created++
			}
		}

	}
	return nil
}

func (n *Nginx) shouldReload() bool {
	return n.CycleStats.Created > 0 || n.CycleStats.Deleted > 0
}

func (n *Nginx) start() error {
	cmd := exec.Command(nginxBinPatch)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	n.logger.
		With(zap.String("output", string(output))).
		Info("nginx started successfully")
	return nil
}

func (n *Nginx) reload() error {
	cmd := exec.Command(nginxBinPatch, "-t")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	n.logger.
		With(zap.String("output", string(output))).
		Info("nginx configs tested successfully")
	cmd = exec.Command(nginxBinPatch, "-s", "reload")
	output, err = cmd.Output()
	if err != nil {
		return err
	}
	n.logger.
		With(zap.String("output", string(output))).
		Info("nginx configs reloaded successfully")

	return nil
}

func (s *ConfigState) exists(desiredState []*ConfigState) bool {
	for _, desired := range desiredState {
		if s.Checksum == desired.Checksum {
			return true
		}
	}
	return false
}

func (s *ConfigState) fileName() string {
	return fmt.Sprintf("/%s/%d-%s.conf", s.BasePath, s.ConfigId, s.Checksum)
}

func (s *ConfigState) remove() error {
	return os.Remove(s.fileName())
}

func (s *ConfigState) create() error {
	return os.WriteFile(s.fileName(), s.Spec, 0644)
}

func (s *CycleStats) reset() {
	s.Created = 0
	s.Deleted = 0
}
