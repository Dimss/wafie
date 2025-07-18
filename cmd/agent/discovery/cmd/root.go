package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "wafie-discovery-agent",
		Short: "WAFie Discovery Agent",
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
		// setup logging
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logger, _ := config.Build()
		zap.ReplaceGlobals(logger)
		// setup viper
		viper.AutomaticEnv()
		viper.SetEnvPrefix("CWAF_DISCOVERY_AGENT")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	})
}
