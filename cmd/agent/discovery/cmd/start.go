package cmd

import (
	"fmt"
	"github.com/Dimss/wafie/pkg/agent/discovery/ingresscache"
	hsrv "github.com/Dimss/wafie/pkg/healthchecksrv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	startCmd.PersistentFlags().StringP(
		"ingress-type",
		"i",
		"ingress",
		fmt.Sprintf("one of %s|%s|%s",
			ingresscache.VsIngressType,
			ingresscache.K8sIngressType,
			ingresscache.RouteIngressType),
	)
	startCmd.PersistentFlags().StringP("api-addr", "a", "http://localhost:8080", "API address")
	viper.BindPFlag("ingress-type", startCmd.PersistentFlags().Lookup("ingress-type"))
	viper.BindPFlag("api-addr", startCmd.PersistentFlags().Lookup("api-addr"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start wafie discovery agent",
	Run: func(cmd *cobra.Command, args []string) {
		// start health check server
		hsrv.NewHealthCheckServer(
			":8081", viper.GetString("api-addr"),
		).Serve()
		// start ingress cache
		cache := ingresscache.NewIngressCache(
			viper.GetString("ingress-type"),
			viper.GetString("api-addr"))
		cache.Start()
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
