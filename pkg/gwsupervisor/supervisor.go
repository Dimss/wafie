package gwsupervisor

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"time"
)

type Supervisor struct {
	envoyPath           string
	envoyConfigFile     string
	logrotatePath       string
	logrotateConfigFile string
	logger              *zap.Logger
}

func NewGatewaySupervisor(log *zap.Logger) *Supervisor {
	return &Supervisor{
		envoyPath:           "/usr/local/bin/envoy",
		envoyConfigFile:     "/etc/envoy/envoy-xds.yaml",
		logrotatePath:       "/usr/sbin/logrotate",
		logrotateConfigFile: "/etc/envoy/wafie-logrotate.conf",
		logger:              log,
	}
}

func (s *Supervisor) Start() {
	s.startGatewayProcess()
	s.startLogRotationProcess()
}

func (s *Supervisor) startGatewayProcess() {
	s.logger.Info("starting WAFie Gateway Supervisor")
	s.runBackgroundCmd(
		exec.Command(
			s.envoyPath, []string{"-c", s.envoyConfigFile}...,
		),
	)
}

func (s *Supervisor) startLogRotationProcess() {
	go func() {
		s.logger.Info("starting log rotation process")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.logger.Info("executing rotation")
				s.runBackgroundCmd(
					exec.Command(
						s.logrotatePath, []string{"-f", s.logrotateConfigFile}...,
					),
				)
			}
		}
	}()
}

func (s *Supervisor) runBackgroundCmd(cmd *exec.Cmd) {
	reader, writer := io.Pipe()
	cmd.Stdin = os.Stdin
	cmd.Stdout = writer
	cmd.Stderr = writer
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			msg := scanner.Text()
			fmt.Println(msg)
		}
	}()
	go func() {
		if err := cmd.Run(); err != nil {
			s.logger.Error(
				"error running command",
				zap.Strings("args", cmd.Args),
				zap.Error(err),
			)
		}
	}()
}
