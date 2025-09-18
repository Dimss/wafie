package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dimss/wafie/relay/pkg/nftables"
	"github.com/Dimss/wafie/relay/pkg/relay"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	startCmd.PersistentFlags().StringP("netns", "n", "", "Network namespace mount path")
	viper.BindPFlag("netns", startCmd.PersistentFlags().Lookup("netns"))
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the relay",
	Run: func(cmd *cobra.Command, args []string) {
		errChan := make(chan error)
		relay := relay.New(errChan)
		// define all the executions
		run := func() {
			errChan <- nftables.Apply()
			go relay.Run()
			shutdown(relay, errChan)
		}
		// netns not set, executing in current namespace
		if viper.GetString("netns") == "" {
			log.Println("network namespace not set")
			run()
		} else {
			// netns is set, entering the namespace and executing
			netNs, err := ns.GetNS(viper.GetString("netns"))
			if err != nil {
				errChan <- err
			}
			defer netNs.Close()
			_ = netNs.Do(func(_ ns.NetNS) error {
				log.Printf("network namespace set: %s\n", viper.GetString("netns"))
				run()
				return nil
			})
		}
	},
}

func start() {

}

func shutdown(relay *relay.Relay, errChan chan error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("error: %v", err)
				sigCh <- syscall.SIGTERM
			}
		case sig := <-sigCh:
			relay.Stop()
			log.Printf("signal received, shutting down with singnal: %s, bye bye 👋\n", sig.String())
			if s, ok := sig.(syscall.Signal); ok {
				os.Exit(128 + int(s))
			}
			os.Exit(1)
		}
	}
}

func main() {
	if err := startCmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
