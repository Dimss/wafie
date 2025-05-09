package cmd

import (
	"github.com/Dimss/cwaf/internal/applogger"
	"github.com/Dimss/cwaf/internal/models"
	"github.com/Dimss/cwaf/pkg/apiserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	//"go.uber.org/zap"
	//"go.uber.org/zap/zapcore"
	//"gorm.io/gorm/logger"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	startCmd.PersistentFlags().StringP("db-host", "", "localhost", "Database host")
	startCmd.PersistentFlags().IntP("db-port", "", 5432, "Database port")
	startCmd.PersistentFlags().StringP("db-user", "", "cwafpg", "Database user")
	startCmd.PersistentFlags().StringP("db-password", "", "cwafpg", "Database password")
	startCmd.PersistentFlags().StringP("db-name", "", "cwaf", "Database name")

	viper.BindPFlag("db-host", startCmd.PersistentFlags().Lookup("db-host"))
	viper.BindPFlag("db-port", startCmd.PersistentFlags().Lookup("db-port"))
	viper.BindPFlag("db-user", startCmd.PersistentFlags().Lookup("db-user"))
	viper.BindPFlag("db-password", startCmd.PersistentFlags().Lookup("db-password"))
	viper.BindPFlag("db-name", startCmd.PersistentFlags().Lookup("db-name"))

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start api server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := applogger.NewLogger()
		defer logger.Sync()
		logger.Info("starting api server")
		_, err := models.NewDb(
			models.NewDbCfg(
				viper.GetString("db-host"),
				viper.GetInt("db-port"),
				viper.GetString("db-user"),
				viper.GetString("db-password"),
				viper.GetString("db-name"),
				logger,
			),
		)
		if err != nil {
			logger.Fatal("error during database connection initialization", zap.Error(err))
		}
		srv := apiserver.NewApiServer(logger)
		srv.Start()

		// handle interrupts
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case s := <-sigCh:
				logger.Info("signal received, shutting down", zap.String("signal", s.String()))
				logger.Info("bye bye 👋")
				os.Exit(0)
			}
		}
	},
}
