package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	startCmd.PersistentFlags().BoolP("log-to-file", "l", true, "Log to file instead of stdout")
	viper.BindPFlag("log-to-file", startCmd.PersistentFlags().Lookup("log-to-file"))

}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the network namespace discovery agent",
	Run: func(cmd *cobra.Command, args []string) {
		logToFile()
		netId, err := pidNsRefToNetId("/proc/557751/ns/net")
		if err != nil {
			log.Printf("error: %v,\n", err)
		} else {
			if err := netIdToNamedNetworkNamespace(netId); err != nil {
				log.Printf("error: %v,\n", err)
			}
		}
	},
}

func pidNsRefToNetId(pidNsRef string) (uint64, error) {
	entry, err := os.Readlink(pidNsRef)
	if err != nil {
		return 0, err
	}
	r, err := regexp.Compile("\\d")
	if err != nil {
		return 0, err
	}
	res := strings.Join(r.FindAllString(entry, -1), "")
	return strconv.ParseUint(res, 10, 64)

}

func netIdToNamedNetworkNamespace(netId uint64) error {
	path := "/var/run/netns"
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		stat := info.Sys().(*syscall.Stat_t)
		if netId == stat.Ino {
			log.Println(fullPath)
		}

	}

	return nil
}

func logToFile() {
	if viper.GetBool("log-to-file") {
		log.SetOutput(
			&lumberjack.Logger{
				Filename:   "relay.log",
				MaxSize:    5,
				MaxBackups: 1,
				MaxAge:     3,
			},
		)
	}
}

func main() {
	if err := startCmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
