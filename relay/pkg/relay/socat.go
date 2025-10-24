package relay

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"

	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"go.uber.org/zap"
)

type Socat struct {
	cmd     *exec.Cmd
	command string
	args    []string
	cancel  context.CancelFunc
	logger  *zap.Logger
}

func NewSocat(logger *zap.Logger) *Socat {
	return &Socat{logger: logger}
}

func (r *Socat) Start(options *wv1.RelayOptions) {
	// relay already running, do nothing
	if r.cmd != nil && r.cmd.Process != nil && r.cmd.ProcessState == nil {
		return
	}
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	if len(options.ProxyIps) < 1 {
		r.logger.Error("can not start relay, appsecgw ips list is empty is empty")
		return
	}
	r.cmd = exec.CommandContext(ctx,
		"socat",
		"-d",
		fmt.Sprintf("TCP-LISTEN:%s,"+
			"reuseaddr,fork,backlog=2048,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack",
			options.RelayPort),
		fmt.Sprintf("TCP:%s:%s,"+
			"rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack,connect-timeout=3",
			options.ProxyIps[0], options.ProxyListeningPort),
	)
	go func() {
		r.setupLogs()
		if err := r.cmd.Start(); err != nil {
			r.logger.Error("socat start error", zap.Error(err))
		}
		if err := r.cmd.Wait(); err != nil {
			r.logger.Error("socat run error", zap.Error(err))
		}
	}()
}

func (r *Socat) Stop(_ *wv1.RelayOptions) {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Socat) Status() {}

func (r *Socat) setupLogs() {
	stdout, _ := r.cmd.StdoutPipe()
	stderr, _ := r.cmd.StderrPipe()
	go readProgramOutput(stdout)
	go readProgramOutput(stderr)
}

func readProgramOutput(readCloser io.ReadCloser) {
	_, err := io.Copy(log.Writer(), readCloser)
	if err != nil {
		log.Printf("error: %v", err)
	}
}
