package controlplane

import (
	"fmt"
	"time"

	cwafv1 "github.com/Dimss/wafie/api/gen/wafie/v1"
	"github.com/Dimss/wafie/internal/applogger"
	golangv3alpha "github.com/envoyproxy/go-control-plane/contrib/envoy/extensions/filters/http/golang/v3alpha"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	stream "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

type state struct {
	logger *zap.Logger
}

func newState() *state {
	return &state{
		logger: applogger.NewLogger(),
	}
}

func (s *state) httpFilters() []*hcm.HttpFilter {
	totalFilters := 2 // router + custom wafie
	var filters = make([]*hcm.HttpFilter, totalFilters)

	wafieLibCfg, err := anypb.New(&golangv3alpha.Config{
		LibraryId:   "wafie-v1",
		LibraryPath: "/usr/local/lib/wafie-modsec.so",
		PluginName:  "wafie",
	})
	if err != nil {
		s.logger.Error("failed to create wafie config", zap.Error(err))
	}
	filters[0] = &hcm.HttpFilter{
		Name: "envoy.filters.http.golang",
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: wafieLibCfg,
		},
	}
	routerConfig, err := anypb.New(&router.Router{})
	if err != nil {
		s.logger.Error("failed to create router config", zap.Error(err))
	}
	filters[1] = &hcm.HttpFilter{
		Name: wellknown.Router,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: routerConfig,
		},
	}
	return filters
}

func (s *state) httpConnectionManager(protection *cwafv1.Protection) *hcm.HttpConnectionManager {
	stdoutLogs, _ := anypb.New(&stream.StdoutAccessLog{})
	return &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		GenerateRequestId: &wrappers.BoolValue{
			Value: true,
		},
		AccessLog: []*accesslog.AccessLog{
			{
				Name: "envoy.access_loggers.stdout",
				ConfigType: &accesslog.AccessLog_TypedConfig{
					TypedConfig: stdoutLogs,
				},
			},
		},
		HttpFilters: s.httpFilters(),
		UpgradeConfigs: []*hcm.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: "websocket",
			},
		},
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				Name: "local_route",
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    protection.Application.Name,
						Domains: []string{protection.Application.Ingress[0].Host},
						Routes: []*route.Route{
							{
								Name: protection.Application.Name,
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										Timeout: durationpb.New(0 * time.Second), // zero meaning disabled
										ClusterSpecifier: &route.RouteAction_Cluster{
											Cluster: protection.Application.Name,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *state) listeners(protections []*cwafv1.Protection) []types.Resource {
	var listeners = make([]types.Resource, len(protections))
	for i := 0; i < len(protections); i++ {
		httpConnectionMgr, _ := anypb.New(s.httpConnectionManager(protections[i]))
		listeners[i] = &v3listener.Listener{
			Name: fmt.Sprintf("listener-%d", i),
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(protections[i].Application.Ingress[0].ProxyListenerPort),
						},
					},
				},
			},
			FilterChains: []*v3listener.FilterChain{
				{
					Filters: []*v3listener.Filter{
						{
							Name: wellknown.HTTPConnectionManager,
							ConfigType: &v3listener.Filter_TypedConfig{
								TypedConfig: httpConnectionMgr,
							},
						},
					},
				},
			}}
	}
	return listeners
}

func (s *state) clusters(protections []*cwafv1.Protection) (clusters []types.Resource) {
	clusters = make([]types.Resource, 0, len(protections))
	for _, protection := range protections {
		if shouldSkipProtection(protection) {
			continue
		}
		clusters = append(clusters, &cluster.Cluster{
			Name:                 protection.Application.Name,
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
			ConnectTimeout:       durationpb.New(20 * time.Second),
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      cluster.Cluster_V4_ONLY,
			LoadAssignment: &endpoint.ClusterLoadAssignment{
				ClusterName: protection.Application.Name,
				Endpoints: []*endpoint.LocalityLbEndpoints{
					{
						LbEndpoints: []*endpoint.LbEndpoint{
							{
								HostIdentifier: &endpoint.LbEndpoint_Endpoint{
									Endpoint: &endpoint.Endpoint{
										Address: &core.Address{
											Address: &core.Address_SocketAddress{
												SocketAddress: &core.SocketAddress{
													Protocol: core.SocketAddress_TCP,
													Address:  protection.Application.Ingress[0].UpstreamHost,
													PortSpecifier: &core.SocketAddress_PortValue{
														PortValue: uint32(
															protection.Application.Ingress[0].UpstreamPort,
														),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
	}
	return clusters
}

func (s *state) buildResources(protections []*cwafv1.Protection) map[resource.Type][]types.Resource {
	return map[resource.Type][]types.Resource{
		resource.ListenerType: s.listeners(protections),
		resource.ClusterType:  s.clusters(protections),
	}
}
