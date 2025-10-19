package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/appsecgw/pkg/controlplane"
	"github.com/Dimss/wafie/logger"
	"go.uber.org/zap"

	hsrv "github.com/Dimss/wafie/apisrv/pkg/healthchecksrv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	startCmd.PersistentFlags().StringP("api-addr", "a", "http://localhost:8080", "API address")
	startCmd.PersistentFlags().StringP("namespace", "n", "default", "K8s namespace")
	viper.BindPFlag("api-addr", startCmd.PersistentFlags().Lookup("api-addr"))
	viper.BindPFlag("namespace", startCmd.PersistentFlags().Lookup("namespace"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Wafie AppSec Gateway control plane envoy gRPC server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logger.NewLogger()
		// start health check server
		hsrv.NewHealthCheckServer(
			":8082", viper.GetString("api-addr"),
		).Serve()
		logger.Info("starting AppSec Gateway gRPC server")
		go controlplane.
			NewEnvoyControlPlane(
				viper.GetString("api-addr"),
				viper.GetString("namespace"),
			).Start()
		// start envoy proxy and modsec (wafie-modsec.so) log rotation
		go controlplane.
			NewSupervisor(logger).
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
