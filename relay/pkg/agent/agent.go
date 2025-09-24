package agent

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	wafiev1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/api/gen/wafie/v1/wafiev1connect"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Protections struct {
	logger              *zap.Logger
	protectionSvcClient wafiev1connect.ProtectionServiceClient
	protectionsChannel  chan []*wafiev1.Protection
}

func NewProtections(apiAddr string, logger *zap.Logger) *Protections {
	return &Protections{
		logger:             logger,
		protectionsChannel: make(chan []*wafiev1.Protection, 1),
		protectionSvcClient: wafiev1connect.NewProtectionServiceClient(
			http.DefaultClient, apiAddr,
		),
	}
}

func (r *Protections) Run() {
	r.startProtectionsPooling()
	r.relayInstancesMgr()
}

func (r *Protections) startProtectionsPooling() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			mode := wafiev1.ProtectionMode_PROTECTION_MODE_ON
			includeApps := true
			req := connect.NewRequest(&wafiev1.ListProtectionsRequest{
				Options: &wafiev1.ListProtectionsOptions{
					ProtectionMode: &mode,
					IncludeApps:    &includeApps,
				},
			})
			resp, err := r.protectionSvcClient.ListProtections(context.Background(), req)
			if err != nil {
				r.logger.Error("failed to list protections", zap.Error(err))
				continue
			}
			r.protectionsChannel <- resp.Msg.Protections
		}
	}()
}

func (r *Protections) relayInstancesMgr() {
	go func() {
		for protections := range r.protectionsChannel {
			for _, protection := range protections {
				r.logger.Info(protection.Application.Name)
			}
		}
	}()
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

	log.SetOutput(
		&lumberjack.Logger{
			Filename:   "relay.log",
			MaxSize:    5,
			MaxBackups: 1,
			MaxAge:     3,
		},
	)

}
