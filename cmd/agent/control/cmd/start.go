package cmd

import (
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/Dimss/cwaf/pkg/agent/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	startCmd.PersistentFlags().
		StringP("api-addr", "", "http://localhost:8080", "API address")
	startCmd.PersistentFlags().
		StringP("nginx-vs-path", "",
			"/opt/app/nginx/conf/protected-services", "path to nginx virtual server config")
	viper.BindPFlag("api-addr", startCmd.PersistentFlags().Lookup("api-addr"))
	viper.BindPFlag("nginx-vs-path", startCmd.PersistentFlags().Lookup("nginx-vs-path"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "StartCycleLoop the agent",
	Run: func(cmd *cobra.Command, args []string) {
		nginxController := controller.NewNginxController(
			viper.GetString("nginx-vs-path"),
			applogger.NewLogger(),
			cwafv1connect.NewVirtualHostServiceClient(
				&http.Client{},
				viper.GetString("api-addr"),
			))

		go func() {
			nginxController.StartCycleLoop()
		}()

		// handle interrupts
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case s := <-sigCh:
				zap.S().Infof("signal: %s, shutting down", s)
				zap.S().Info("bye bye 👋")
				os.Exit(0)
			}
		}

	},
}
