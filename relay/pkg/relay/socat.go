package relay

import (
	"context"
	"io"
	"log"
	"os/exec"
)

type Socat struct {
	cmd     *exec.Cmd
	command string
	args    []string
	cancel  context.CancelFunc
	errChan chan error
}

func NewSocat(errChan chan error) *Socat {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx,
		"socat",
		"-d",
		"TCP-LISTEN:9090,reuseaddr,fork,backlog=2048,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack",
		"TCP:10.244.0.12:8888,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack,connect-timeout=3")
	return &Socat{
		cmd:     cmd,
		cancel:  cancel,
		errChan: errChan,
	}
}

func (r *Socat) Start() {
	r.setupLogs()
	if err := r.cmd.Start(); err != nil {
		log.Printf("failed to start command: %v\n", err)
		r.errChan <- err
	}
	r.errChan <- r.cmd.Wait()
}

func (r *Socat) Stop() {
	r.cancel()
}

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
