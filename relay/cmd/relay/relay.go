package relay

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/internal/applogger"
	"github.com/Dimss/wafie/relay/pkg/apisrv"
	"github.com/Dimss/wafie/relay/pkg/nftables"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger = applogger.NewLoggerToFile()
	startCmd.AddCommand(relayCmd)
}

var relayCmd = &cobra.Command{
	Use:   "relay-instance",
	Short: "start wafie relay instance",
	Run: func(cmd *cobra.Command, args []string) {
		errChan := make(chan error)
		socatRelay := relay.NewSocat(errChan)
		// start relay api server
		apisrv.
			NewServer("localhost:8081", logger, socatRelay).
			Serve()
		// Program NFTables
		go nftables.Program(errChan)
		// Start TCP relay
		go socatRelay.Start()
		// gracefully wait for shutdown
		shutdown(socatRelay, errChan)
	},
}

func shutdown(s relay.Relay, errChan chan error) {
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
		case err := <-errChan:
			if err != nil {
				logger.Error("received an error on errChan", zap.Error(err))
				gracefullyExit(s, os.Signal(syscall.SIGABRT))
			}
		case sig := <-sigCh:
			gracefullyExit(s, sig)
		}
	}
}
