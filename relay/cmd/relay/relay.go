package relay

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/internal/applogger"
	"github.com/Dimss/wafie/relay/pkg/apisrv"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	relayCmd.PersistentFlags().BoolP("logs-to-stdout", "l", false, "Print logs to stdout instead of file")
	viper.BindPFlag("logs-to-stdout", relayCmd.PersistentFlags().Lookup("logs-to-stdout"))
	startCmd.AddCommand(relayCmd)
}

var relayCmd = &cobra.Command{
	Use:   "relay-instance",
	Short: "start wafie relay instance",
	Run: func(cmd *cobra.Command, args []string) {
		logger := initLogger()
		socatRelay := relay.NewSocat(logger)
		// start relay api server
		apisrv.
			NewServer("localhost:8081", logger, socatRelay).
			Start()
		shutdown(socatRelay)
	},
}

func initLogger() *zap.Logger {
	if viper.GetBool("logs-to-stdout") {
		return applogger.NewLogger()
	}
	return applogger.NewLoggerToFile()
}

func shutdown(s relay.Relay) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	gracefullyExit := func(s relay.Relay, sig os.Signal) {
		s.Stop()
		logger.Info("shutting down, bye bye 👋", zap.String("signal", sig.String()))
		if s, ok := sig.(syscall.Signal); ok {
			os.Exit(128 + int(s))
		}
		os.Exit(1)
	}
	for {
		select {
		//case err := <-errChan:
		//	if err != nil {
		//		logger.Error("received an error on errChan", zap.Error(err))
		//		gracefullyExit(s, os.Signal(syscall.SIGTERM))
		//	}
		case sig := <-sigCh:
			gracefullyExit(s, sig)
		}
	}
}
