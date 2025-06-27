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
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"k8s.io/client-go/kubernetes"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type EnvoyControlPlane struct {
	state                *state
	cache                cache.SnapshotCache
	logger               *zap.Logger
	resourcesCh          chan map[resource.Type][]types.Resource
	ingressPatcherCh     chan []*cwafv1.Protection
	dataVersion          string
	namespace            string
	protectionSvcClient  cwafv1connect.ProtectionServiceClient
	dataVersionSvcClient cwafv1connect.DataVersionServiceClient
}

func NewEnvoyControlPlane(apiAddr, namespace string) *EnvoyControlPlane {

	cp := &EnvoyControlPlane{
		state:            newState(),
		logger:           applogger.NewLogger(),
		resourcesCh:      make(chan map[resource.Type][]types.Resource, 1),
		ingressPatcherCh: make(chan []*cwafv1.Protection, 1),
		namespace:        namespace,
		cache: cache.NewSnapshotCache(
			false, cache.IDHash{}, applogger.NewLogger().Sugar(),
		),
		protectionSvcClient: cwafv1connect.NewProtectionServiceClient(
			http.DefaultClient, apiAddr,
		),
		dataVersionSvcClient: cwafv1connect.NewDataVersionServiceClient(
			http.DefaultClient, apiAddr,
		),
	}
	// start control plane data watcher
	cp.startControlPlaneDataWatcher()
	// start envoy snapshot generator
	cp.startSnapshotGenerator()
	// start ingress patcher
	// TODO: this is a hack, need to understand how to properly handle this if at all
	cp.ingressPatcher()
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

func (p *EnvoyControlPlane) dataVersionChanged() bool {
	dataVersionResponse, err := p.dataVersionSvcClient.GetDataVersion(
		context.Background(),
		connect.NewRequest(
			&cwafv1.GetDataVersionRequest{
				TypeId: cwafv1.DataTypeId_DATA_TYPE_ID_PROTECTION,
			},
		),
	)
	if err != nil {
		p.logger.Error("failed to get protection data version", zap.Error(err))
		return false
	}
	// check if the protection data version has changed since last iteration
	if dataVersionResponse.Msg.VersionId == p.dataVersion {
		return false
	}
	p.logger.Info("protection data version has changed",
		zap.String("versionId", dataVersionResponse.Msg.VersionId))
	p.dataVersion = dataVersionResponse.Msg.VersionId
	return true
}

func (p *EnvoyControlPlane) startControlPlaneDataWatcher() {
	p.logger.Info("starting control plane data watcher")
	go func() {
		for {
			time.Sleep(1 * time.Second)
			if !p.dataVersionChanged() {
				continue
			}
			mode := cwafv1.ProtectionMode_PROTECTION_MODE_ON
			includeApps := true
			req := connect.NewRequest(&cwafv1.ListProtectionsRequest{
				Options: &cwafv1.ListProtectionsOptions{
					ProtectionMode: &mode,
					IncludeApps:    &includeApps,
				},
			})
			resp, err := p.protectionSvcClient.ListProtections(context.Background(), req)
			if err != nil {
				p.logger.Error("failed to list protections", zap.Error(err))
				continue
			}
			p.logger.Info("data version has changed, building new resources")
			p.ingressPatcherCh <- resp.Msg.Protections
			p.resourcesCh <- p.state.buildResources(resp.Msg.Protections)

		}
	}()
}

func (p *EnvoyControlPlane) startSnapshotGenerator() {
	p.logger.Info("starting envoy snapshot generator")
	go func() {
		for resources := range p.resourcesCh {
			p.logger.Info("state has been changed, generating new snapshot...")
			snap, _ := cache.NewSnapshot(fmt.Sprintf("%d", rand.Int()), resources)
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

func (p *EnvoyControlPlane) ingressPatcher() {
	p.logger.Info("starting ingress patcher")
	go func(kc *kubernetes.Clientset, kcError error) {
		for protections := range p.ingressPatcherCh {
			if kcError != nil {
				p.logger.Error("kube client in error, can't patch ingresses", zap.Error(kcError))
				continue
			}
			for _, protection := range protections {
				if protection.IngressAutoPatch == cwafv1.IngressAutoPatch_INGRESS_AUTO_PATCH_ON {
					if err := NewIngressPatcher(kc, protection, p.namespace, p.logger).Patch(); err != nil {
						p.logger.Error("failed to patch ingress", zap.Error(err))
					}
				}
				if protection.IngressAutoPatch == cwafv1.IngressAutoPatch_INGRESS_AUTO_PATCH_OFF {
					if err := NewIngressPatcher(kc, protection, p.namespace, p.logger).Unpatch(); err != nil {
						p.logger.Error("failed to unpatch ingress", zap.Error(err))
					}
				}
			}
		}
	}(newKubeClient())
}
