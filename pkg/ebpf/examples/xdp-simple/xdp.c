//go:build ignore

#include "bpf_endian.h"
#include "common.h"

char _license[] SEC("license") = "GPL";

#define MAX_MAP_ENTRIES 128

/* Define an LRU hash map for storing packet count by source IPv4 address */
struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, MAX_MAP_ENTRIES);
	__type(key, __u32); // source IPv4 address
	__type(value, __u32); // packet count
} xdp_stats_map SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, MAX_MAP_ENTRIES);
	__type(key, __u32); // source IPv4 address
	__type(value, __u32); // packet count
} xdp_stats_map2 SEC(".maps");

//struct {
//    __uint(type, BPF_MAP_TYPE_ARRAY);
//    __uint(max_entries, 1);
//    __type(key, __u32);
//    __type(value, __u32); // e.g., an IP to block
//} config_map SEC(".maps");

static __always_inline int parse_ip_src_addr(struct xdp_md *ctx, __u32 *ip_src_addr, __u32 *ip_dst_addr) {
	void *data_end = (void *)(long)ctx->data_end;
	void *data     = (void *)(long)ctx->data;

	// First, parse the ethernet header.
	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end) {
		return 0;
	}

	if (eth->h_proto != bpf_htons(ETH_P_IP)) {
		// The protocol is not IPv4, so we can't parse an IPv4 source address.
		return 0;
	}

	// Then parse the IP header.
	struct iphdr *ip = (void *)(eth + 1);
	if ((void *)(ip + 1) > data_end) {
		return 0;
	}

	// Return the source IP address in network byte order.
	*ip_src_addr = (__u32)(ip->saddr);
	// Return the destination IP address in network byte order.
	*ip_dst_addr = (__u32)(ip->daddr);
	return 1;
}

SEC("xdp")
int xdp_prog_func(struct xdp_md *ctx) {
    __u32 src_ip;
    __u32 dst_ip;
    if (!parse_ip_src_addr(ctx, &src_ip, &dst_ip)){
        goto done;
    }

//    __u32 key = 0;
//    __u32 *blocked_ip;
//    blocked_ip = bpf_map_lookup_elem(&config_map, &key);
//    if (!blocked_ip || *blocked_ip == 0) {
//        // If no parameter is set, don't do anything.
//        return done;
//    }


    __u32 *pkt_count = bpf_map_lookup_elem(&xdp_stats_map, &src_ip);
    if (!pkt_count){
        __u32 init_pkt_count = 1;
        bpf_map_update_elem(&xdp_stats_map, &src_ip, &init_pkt_count, BPF_ANY);
    }else{
        __sync_fetch_and_add(pkt_count, 1);
    }

    __u32 *pkt_count2 = bpf_map_lookup_elem(&xdp_stats_map2, &dst_ip);
    if (!pkt_count2){
        __u32 init_pkt_count2 = 1;
        bpf_map_update_elem(&xdp_stats_map2, &dst_ip, &init_pkt_count2, BPF_ANY);
    }else{
        __sync_fetch_and_add(pkt_count2, 1);
    }

done:
	return XDP_PASS;
}