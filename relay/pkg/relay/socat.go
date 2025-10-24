package relay

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os/exec"
	"time"

	wv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"go.uber.org/zap"
)

type SocatRelay struct {
	cmd     *exec.Cmd
	command string
	args    []string
	cancel  context.CancelFunc
	logger  *zap.Logger
	options *wv1.RelayOptions
}

func NewSocat(logger *zap.Logger) *SocatRelay {
	return &SocatRelay{logger: logger}
}

func (r *SocatRelay) shouldRestart(cfgOptions *wv1.RelayOptions) bool {
	if cfgOptions == nil {
		return false
	}
	return r.options.ProxyFqdn != cfgOptions.ProxyFqdn ||
		r.options.AppContainerPort != cfgOptions.AppContainerPort ||
		r.options.RelayPort != cfgOptions.RelayPort
}

func (r *SocatRelay) Configure(cfgOptions *wv1.RelayOptions) (StartRelayFunc, StopRelayFunc) {
	// if options nil, meaning fresh start
	if r.options == nil {
		r.options = cfgOptions
		r.setProxyIp()
		return r.start, r.stop
	}
	// if options already set, meaning relay already running
	// if start options changed from previous start,
	// the options must re-initiated
	// and relay must be restarted
	if r.shouldRestart(cfgOptions) {
		r.options = cfgOptions
		r.setProxyIp()
		r.stop()
	}
	return r.start, r.stop
}

func (r *SocatRelay) setProxyIp() {
	if r.options != nil && r.options.ProxyFqdn != "" {
		ips, err := net.LookupHost(r.options.ProxyFqdn)
		if err != nil {
			r.logger.Error("failed to set proxy ip", zap.Error(err), zap.String("proxyFqdn", r.options.ProxyFqdn))
			return
		}
		if len(ips) == 0 {
			r.logger.Error("empty IPs for", zap.String("proxyFqdn", r.options.ProxyFqdn))
			return
		}
		if len(ips) == 1 {
			r.options.ProxyIp = ips[0]
			return
		}
		rand.NewSource(time.Now().UnixNano())
		// the r.options.ProxyFqdn is usually K8s headless svc
		// which has behind multiple A records (pods IPs)
		// thus, I am just implementing
		// simple client side load balancing
		r.options.ProxyIp = ips[rand.Intn(len(ips))]
	} else {
		r.logger.Debug("empty relay options, can not make dns lookup")
	}
}

func (r *SocatRelay) start() {
	if r.cmd != nil && r.cmd.Process != nil && r.cmd.ProcessState == nil {
		return
	}
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	r.cmd = exec.CommandContext(ctx,
		"socat",
		"-d",
		fmt.Sprintf("TCP-LISTEN:%s,"+
			"reuseaddr,fork,backlog=2048,rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack",
			r.options.RelayPort),
		fmt.Sprintf("TCP:%s:%s,"+
			"rcvbuf=262144,sndbuf=262144,keepalive,nodelay,quickack,connect-timeout=3",
			r.options.ProxyIp, r.options.ProxyListeningPort),
	)
	go func() {
		r.setupLogs()
		if err := r.cmd.Start(); err != nil {
			r.logger.Error("socat start error", zap.Error(err))
		}
		if err := r.setupNetwork(); err != nil {
			r.logger.Error("failed to setup network rules", zap.Error(err))
		}
		if err := r.cmd.Wait(); err != nil {
			r.logger.Error("socat run error", zap.Error(err))
		}
	}()
}

func (r *SocatRelay) setupNetwork() error {
	return ProgramNft(AddOp, r.options)
}

func (r *SocatRelay) stop() {
	if r.cancel != nil {
		r.cancel()
	}
	_ = ProgramNft(DeleteOp, r.options)
}

func (r *SocatRelay) Status() {}

func (r *SocatRelay) setupLogs() {
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
