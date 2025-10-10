package relay

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/relay/pkg/nftables"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"github.com/spf13/cobra"
)

func init() {
	//log.SetOutput(
	//	&lumberjack.Logger{
	//		Filename:   "relay.log",
	//		MaxSize:    5,
	//		MaxBackups: 1,
	//		MaxAge:     3,
	//	},
	//)
	//relayCmd.PersistentFlags().StringP("netns", "n", "", "Network namespace mount path")
	//viper.BindPFlag("netns", relayCmd.PersistentFlags().Lookup("netns"))
	startCmd.AddCommand(relayCmd)
}

var relayCmd = &cobra.Command{
	Use:   "relay-instance",
	Short: "start wafie relay instance",
	Run: func(cmd *cobra.Command, args []string) {
		errChan := make(chan error)
		relay := relay.New(errChan)
		// Program NFTables
		go nftables.Program(errChan)
		// Start TCP relay
		go relay.Run()
		// gracefully wait for shutdown
		shutdown(relay, errChan)
	},
}

func shutdown(r *relay.Relay, errChan chan error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	gracefullyExit := func(r *relay.Relay, sig os.Signal) {
		r.Stop()
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
				gracefullyExit(r, os.Signal(syscall.SIGABRT))
			}
		case sig := <-sigCh:
			gracefullyExit(r, sig)
		}
	}
}
