package relay

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:   "wafie-relay",
		Short: "WAFie Relay Agent",
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
		viper.SetEnvPrefix("WAFIE_RELAY")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
