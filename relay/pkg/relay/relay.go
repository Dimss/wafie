package relay

import (
	"context"
	"io"
	"log"
	"os/exec"
)

type Relay struct {
	cmd     *exec.Cmd
	command string
	args    []string
	cancel  context.CancelFunc
	errChan chan error
}

func New(errChan chan error) *Relay {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "socat", "-dddd", "TCP-LISTEN:9090,fork", "TCP:10.244.0.12:8888")
	return &Relay{
		cmd:     cmd,
		cancel:  cancel,
		errChan: errChan,
	}
}

func (r *Relay) Run() {
	r.setupLogs()
	if err := r.cmd.Start(); err != nil {
		log.Printf("failed to start command: %v\n", err)
		r.errChan <- err
	}
	r.errChan <- r.cmd.Wait()
}

func (r *Relay) Stop() {
	r.cancel()
}

func (r *Relay) setupLogs() {
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
