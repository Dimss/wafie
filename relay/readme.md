### WAFie Relay
* Relay instance
* Relay manager


##### Debug Commands
* List all available service: `grpcurl -plaintext localhost:57812 list`
* List all available methods for `RelayService`: `grpcurl -plaintext localhost:57812 list wafie.v1.RelayService` 
* Health check `grpcurl -plaintext localhost:57812 grpc.health.v1.Health/Check`
* Start: `grpcurl -plaintext -d '{"options": {"app_container_port": "8080","appsecgw_ips": ["172.16.0.101"],"appsecgw_listener_port": "52073","relay_port": "50010"}}' localhost:57812 wafie.v1.RelayService.StartRelay`
* Stop: `grpcurl -plaintext localhost:57812 wafie.v1.RelayService.StopRelay`
* Delete nft tables: `nft delete table inet appsecgw`
* List nft tables: `nft list tables`
* List rules: `nft list ruleset`