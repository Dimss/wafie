package relay

import (
	"context"
	"io"
	"log"
	"os/exec"

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

func (r *Socat) Start() {
	// relay already running, do nothing
	if r.cmd != nil && r.cmd.Process != nil && r.cmd.ProcessState == nil {
		return
	}
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	r.cmd = exec.CommandContext(ctx,
		"socat",
		"-d",
		"TCP-LISTEN:9090,reuseaddr,fork,backlog=2048,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack",
		"TCP:172.16.0.101:52073,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack,connect-timeout=3",
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

func (r *Socat) Stop() {
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
