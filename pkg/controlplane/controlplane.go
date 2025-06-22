package controlplane

import (
	"context"
	"fmt"
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
	"os"
	"time"
)

type EnvoyControlPlane struct {
	cache  cache.SnapshotCache
	logger *zap.Logger
}

func NewEnvoyControlPlane() *EnvoyControlPlane {
	s := newState()
	cp := &EnvoyControlPlane{
		cache:  cache.NewSnapshotCache(false, cache.IDHash{}, applogger.NewLogger().Sugar()),
		logger: applogger.NewLogger(),
	}
	snap, _ := cache.NewSnapshot(fmt.Sprintf("%d", rand.Int()), s.resources())
	if err := snap.Consistent(); err != nil {
		cp.logger.Error("snapshot inconsistency", zap.Error(err))
		os.Exit(1)
	}
	if err := cp.cache.SetSnapshot(context.Background(), "node-1", snap); err != nil {
		cp.logger.Error("failed to set snapshot", zap.Error(err))
		os.Exit(1)
	}
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
