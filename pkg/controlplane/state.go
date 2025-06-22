package controlplane

import (
	"fmt"
	"github.com/Dimss/cwaf/internal/applogger"
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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

type state struct {
	//resources map[resource.Type][]types.Resource
	logger *zap.Logger
}

func newState() *state {
	return &state{
		logger: applogger.NewLogger(),
	}
}

func (s state) httpFilters() []*hcm.HttpFilter {
	totalFilters := 2 // router + custom kubeguard
	var filters = make([]*hcm.HttpFilter, totalFilters)

	kubeguardLibCfg, err := anypb.New(&golangv3alpha.Config{
		LibraryId:   "kubeguard-v1",
		LibraryPath: "/go/src/github.com/Dimss/cwaf/kubeguard-modsec.so",
		PluginName:  "kubeguard",
	})
	if err != nil {
		s.logger.Error("failed to create kubeguard config", zap.Error(err))
	}
	filters[0] = &hcm.HttpFilter{
		Name: "envoy.filters.http.golang",
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: kubeguardLibCfg,
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

func (s state) httpConnectionManager() *hcm.HttpConnectionManager {
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
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: "local_route",
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
						ApiConfigSource: &core.ApiConfigSource{
							ApiType:                   core.ApiConfigSource_GRPC,
							TransportApiVersion:       resource.DefaultAPIVersion,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*core.GrpcService{
								{
									TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
										EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
											ClusterName: "pcp_xds_cluster",
										},
									},
								},
							},
						},
					},
					ResourceApiVersion: resource.DefaultAPIVersion,
				},
			},
		},
	}
}

func (s state) listeners() []types.Resource {
	httpConnectionMgr, _ := anypb.New(s.httpConnectionManager())
	return []types.Resource{
		&v3listener.Listener{
			Name: "listener-1",
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 8888,
						},
					},
				}},
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
			},
		},
	}
	//s.resources[resource.ListenerType] = []types.Resource{l}
}

func (s state) clusters(name string) []types.Resource {
	//name := "wp-host"
	return []types.Resource{
		&cluster.Cluster{
			Name:                 name,
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
			ConnectTimeout:       durationpb.New(20 * time.Second),
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      cluster.Cluster_V4_ONLY,
			LoadAssignment: &endpoint.ClusterLoadAssignment{
				ClusterName: name,
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
													Address:  "wp-wordpress.default.svc",
													PortSpecifier: &core.SocketAddress_PortValue{
														PortValue: 80,
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
		},
	}
}

func (s state) routes(name string) []types.Resource {
	//name := "wp-host"
	host := "*"
	return []types.Resource{
		&route.RouteConfiguration{
			Name: "local_route",
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    name,
					Domains: []string{host},
					Routes: []*route.Route{
						{
							Name: name,
							Match: &route.RouteMatch{
								PathSpecifier: &route.RouteMatch_Prefix{
									Prefix: "/",
								},
							},
							Action: &route.Route_Route{
								Route: &route.RouteAction{
									Timeout: durationpb.New(0 * time.Second), // zero meaning disabled
									ClusterSpecifier: &route.RouteAction_Cluster{
										Cluster: name,
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

func (s state) dumpConfigs() {
	configs := ""
	resources := map[resource.Type][]types.Resource{
		resource.ListenerType: s.listeners(),
		resource.ClusterType:  s.clusters("wp-host"),
		resource.RouteType:    s.routes("wp-host"),
	}
	for _, resourceList := range resources {
		for _, res := range resourceList {
			jsonBytes, err := protojson.Marshal(res)
			if err != nil {
				s.logger.Error("failed to marshal resources", zap.Error(err))
				continue
			}
			configs += string(jsonBytes) + "\n"
		}
	}
	fmt.Println(configs)
}

func (s state) resources() map[resource.Type][]types.Resource {

	return map[resource.Type][]types.Resource{
		resource.ListenerType: s.listeners(),
		resource.ClusterType:  s.clusters("wp-host"),
		resource.RouteType:    s.routes("wp-host"),
	}
}
