package controlplane

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	cwafv1 "github.com/Dimss/cwaf/api/gen/cwaf/v1"
	"github.com/Dimss/cwaf/api/gen/cwaf/v1/cwafv1connect"
	"github.com/Dimss/cwaf/internal/applogger"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type EnvoyControlPlane struct {
	state                      *state
	cache                      cache.SnapshotCache
	logger                     *zap.Logger
	notifier                   chan struct{}
	protectionServiceSvcClient cwafv1connect.ProtectionServiceClient
}

func NewEnvoyControlPlane(apiAddr string) *EnvoyControlPlane {

	cp := &EnvoyControlPlane{
		state:    newState(),
		logger:   applogger.NewLogger(),
		notifier: make(chan struct{}, 1),
		cache: cache.NewSnapshotCache(
			false, cache.IDHash{}, applogger.NewLogger().Sugar()),
		protectionServiceSvcClient: cwafv1connect.NewProtectionServiceClient(
			http.DefaultClient, apiAddr),
	}
	// start control plane data watcher
	cp.startControlPlaneDataWatcher()
	// start envoy snapshot generator
	cp.startSnapshotGenerator()
	return cp
}

func (p *EnvoyControlPlane) Start() {
	envoySrv := server.NewServer(context.Background(), p.cache, &test.Callbacks{})
	grpcSrv := grpc.NewServer([]grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  2 * time.Hour,
			Timeout:               20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: false,
		})}...,
	)
	// register the gRPC services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcSrv, envoySrv)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcSrv, envoySrv)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcSrv, envoySrv)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcSrv, envoySrv)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcSrv, envoySrv)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcSrv, envoySrv)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcSrv, envoySrv)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", 18000))
	if err != nil {
		p.logger.Error("failed to listen", zap.Error(err))
	}
	p.logger.Info("Envoy control plane listening started", zap.String("address", lis.Addr().String()))
	if err = grpcSrv.Serve(lis); err != nil {
		zap.S().Fatal(err)
	}
}

func (p *EnvoyControlPlane) startControlPlaneDataWatcher() {
	p.logger.Info("starting control plane data watcher")
	go func() {
		for {
			time.Sleep(1 * time.Second)
			mode := cwafv1.ProtectionMode_PROTECTION_MODE_ON
			req := connect.NewRequest(&cwafv1.ListProtectionsRequest{
				Options: &cwafv1.ListProtectionsOptions{
					ProtectionMode: &mode,
				},
			})
			resp, err := p.protectionServiceSvcClient.ListProtections(context.Background(), req)
			if err != nil {
				p.logger.Error("failed to list protections", zap.Error(err))
				continue
			}
			for _, protection := range resp.Msg.Protections {
				p.logger.Info("protection found",
					zap.Uint32("id", protection.Id),
					zap.String("mode", protection.ProtectionMode.String()),
				)
			}
		}
	}()
}

func (p *EnvoyControlPlane) startSnapshotGenerator() {
	p.logger.Info("starting envoy snapshot generator")
	go func() {
		for _ = range p.notifier {
			p.logger.Info("state has been changed, generating new snapshot...")
			snap, _ := cache.NewSnapshot(fmt.Sprintf("%d", rand.Int()), p.state.resources())
			if err := snap.Consistent(); err != nil {
				p.logger.Error("snapshot inconsistency", zap.Error(err))
				continue
			}
			if err := p.cache.SetSnapshot(context.Background(), "node-1", snap); err != nil {
				p.logger.Error("failed to set snapshot", zap.Error(err))
				continue
			}
		}
	}()
}
