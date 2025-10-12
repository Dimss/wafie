package cmd

import (
	"github.com/Dimss/wafie/internal/applogger"
	"github.com/Dimss/wafie/pkg/gwsupervisor"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

const (
	envoyPath           = "/usr/local/bin/envoy"
	envoyConfigFile     = "/etc/envoy/envoy-xds.yaml"
	logrotatePath       = "/usr/sbin/logrotate"
	logrotateConfigFile = "/tmp/wafie-logrotate.conf"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "StartSpec WAFie Gateway Supervisor",
	Run: func(cmd *cobra.Command, args []string) {
		logger := applogger.NewLogger()
		gwsupervisor.
			NewGatewaySupervisor(logger).
			Start()
		// handle interrupts
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case s := <-sigCh:
				logger.Info("signal received, shutting down", zap.String("signal", s.String()))
				logger.Info("bye bye 👋")
				os.Exit(0)
			}
		}
	},
}
