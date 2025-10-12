### WAFie Relay
* Relay instance
* Relay manager


##### Debug Commands
* List all available service: `grpcurl -plaintext localhost:8081 list`
* List all available methods for `RelayService`: `grpcurl -plaintext localhost:8081 list wafie.v1.RelayService` 
* Health check `grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check`
* Start: `grpcurl -plaintext localhost:8081 wafie.v1.RelayService.StartRelay`
* Stop: `grpcurl -plaintext localhost:8081 wafie.v1.RelayService.StopRelay`
* Delete nft tables: `nft delete table inet wafie-gateway`
* List nft tables: `nft list tables`
* List rules: `nft list ruleset`