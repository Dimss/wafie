package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gwsupervisor",
		Short: "WAFie Gateway Supervisor",
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
		viper.SetEnvPrefix("WAFIE_GWSUPERVISOR")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
