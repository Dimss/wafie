package cmd

import (
	"github.com/Dimss/wafie/internal/applogger"
	"github.com/Dimss/wafie/pkg/controlplane"
	hsrv "github.com/Dimss/wafie/pkg/healthchecksrv"
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
		logger := applogger.NewLogger()
		// start health check server
		hsrv.NewHealthCheckServer(
			":8082", viper.GetString("api-addr"),
		).Serve()
		logger.Info("starting PCP gRPC server")
		controlplane.NewEnvoyControlPlane(
			viper.GetString("api-addr"),
			viper.GetString("namespace"),
		).Start()
	},
}
