package cmd

import (
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/Dimss/cwaf/pkg/controlplane"
	"github.com/spf13/cobra"
)

func init() {

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Proxy Control Plane gRPC server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := applogger.NewLogger()
		defer logger.Sync()
		logger.Info("starting PCP gRPC server")
		envoyControlPlane := controlplane.NewEnvoyControlPlane()
		envoyControlPlane.Start()
	},
}
