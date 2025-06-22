package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "pcp",
		Short: "Proxy Control Plane gRPC Server",
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
		viper.AutomaticEnv()
		viper.SetEnvPrefix("CWAF_PCP_SERVER")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
