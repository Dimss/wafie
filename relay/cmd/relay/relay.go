package relay

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/relay/pkg/apisrv"
	"github.com/Dimss/wafie/relay/pkg/nftables"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	log.SetOutput(
		&lumberjack.Logger{
			Filename:   "relay.log",
			MaxSize:    5,
			MaxBackups: 1,
			MaxAge:     3,
		},
	)
	startCmd.AddCommand(relayCmd)
}

var relayCmd = &cobra.Command{
	Use:   "relay-instance",
	Short: "start wafie relay instance",
	Run: func(cmd *cobra.Command, args []string) {
		errChan := make(chan error)
		socat := relay.NewSocat(errChan)
		// start relay api server
		apisrv.
			NewServer("localhost:8081").
			Serve()
		// Program NFTables
		go nftables.Program(errChan)
		// Start TCP relay
		go socat.Run()
		// gracefully wait for shutdown
		shutdown(socat, errChan)
	},
}

func shutdown(s *relay.Socat, errChan chan error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	gracefullyExit := func(s *relay.Socat, sig os.Signal) {
		s.Stop()
		log.Printf("shutting down with sig: %s, bye bye 👋\n", sig.String())
		if s, ok := sig.(syscall.Signal); ok {
			os.Exit(128 + int(s))
		}
		os.Exit(1)
	}

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("error: %v", err)
				gracefullyExit(s, os.Signal(syscall.SIGABRT))
			}
		case sig := <-sigCh:
			gracefullyExit(s, sig)
		}
	}
}
