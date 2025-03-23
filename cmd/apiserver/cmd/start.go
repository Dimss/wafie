package cmd

import (
	"github.com/Dimss/cwaf/internal/database"
	"github.com/Dimss/cwaf/pkg/apiserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	startCmd.PersistentFlags().StringP("db-host", "", "localhost", "Database host")
	startCmd.PersistentFlags().StringP("db-user", "", "cwafpg", "Database user")
	startCmd.PersistentFlags().StringP("db-password", "", "cwafpg", "Database password")
	startCmd.PersistentFlags().StringP("db-name", "", "cwaf", "Database name")

	viper.BindPFlag("db-host", startCmd.PersistentFlags().Lookup("db-host"))
	viper.BindPFlag("db-user", startCmd.PersistentFlags().Lookup("db-user"))
	viper.BindPFlag("db-password", startCmd.PersistentFlags().Lookup("db-password"))
	viper.BindPFlag("db-name", startCmd.PersistentFlags().Lookup("db-name"))

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start api server",
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("starting api server")
		db, err := database.NewDb(
			database.NewDbCfg(
				viper.GetString("db-host"),
				viper.GetString("db-user"),
				viper.GetString("db-password"),
				viper.GetString("db-name"),
			),
		)
		if err != nil {
			zap.S().Fatalw("error during database connection initialization", "error", err)
		}
		srv := apiserver.NewApiServer(db)
		srv.Start()
		// handle interrupts
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case s := <-sigCh:
				zap.S().Infof("signal: %s, shutting down", s)
				zap.S().Info("bye bye 👋")
				os.Exit(0)
			}
		}
	},
}
