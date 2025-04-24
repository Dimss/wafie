package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "cwaf-control-agent",
		Short: "CWAF Control Agent",
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		// setup viper
		viper.AutomaticEnv()
		viper.SetEnvPrefix("CWAF_CONTROL_AGENT")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
