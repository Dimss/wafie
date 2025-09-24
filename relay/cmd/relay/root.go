package relay

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, _ := config.Build()
		zap.ReplaceGlobals(logger)
		
		viper.AutomaticEnv()
		viper.SetEnvPrefix("WAFIE_RELAY")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
