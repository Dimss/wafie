package cmd

import (
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/Dimss/cwaf/pkg/controlplane"
	hsrv "github.com/Dimss/cwaf/pkg/healthchecksrv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	startCmd.PersistentFlags().StringP("api-addr", "a", "http://localhost:8080", "API address")
	viper.BindPFlag("api-addr", startCmd.PersistentFlags().Lookup("api-addr"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Proxy Control Plane gRPC server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := applogger.NewLogger()
		defer logger.Sync()
		// start health check server
		hsrv.NewHealthCheckServer(
			":8082", viper.GetString("api-addr"),
		).Serve()
		logger.Info("starting PCP gRPC server")
		envoyControlPlane := controlplane.NewEnvoyControlPlane(viper.GetString("api-addr"))
		envoyControlPlane.Start()
	},
}
