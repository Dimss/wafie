//go:build ignore

#include "bpf_endian.h"
#include "common.h"

char __license[] SEC("license") = "Dual MIT/GPL";

#define MAX_MAP_ENTRIES 128

struct ip_pair {
      __be32 src_ip;
      __be32 dst_ip;
};

struct ip_pair_value {
    __u64 count;
    __u32 ifindex;
};


struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, MAX_MAP_ENTRIES);
	__type(key, struct ip_pair);
	__type(value, struct ip_pair_value); // packet count
} ip_map SEC(".maps");



SEC("tc")
int ingress_prog_func(struct __sk_buff *skb) {
	void *data_end = (void *)(long)skb->data_end;
	void *data     = (void *)(long)skb->data;

    // Pointers for parsing
    struct ethhdr *eth = data;
    struct iphdr *iph;

  // Boundary check and ensure it's an IP packet
    if (data + sizeof(*eth) > data_end)
        return TC_ACT_OK;
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    iph = data + sizeof(*eth);
    if ((void *)iph + sizeof(*iph) > data_end)
        return TC_ACT_OK;

    if (iph->protocol != IPPROTO_TCP)
        return TC_ACT_OK;

    struct ip_pair key = {
        .src_ip = iph->saddr,
        .dst_ip = iph->daddr,
    };

    struct ip_pair_value *ipv = bpf_map_lookup_elem(&ip_map, &key);
    if (ipv) {
        __sync_fetch_and_add(&ipv->count, 1);
    } else {
        bpf_printk("iface name: %d", skb->ifindex);
        struct ip_pair_value value = {
            .count = 1,
//            .count2 = 1,
//            .foo = "",
//            .ifindex = skb->ifindex
//            .ifindex = 1
        };
        bpf_map_update_elem(&ip_map, &key, &value, BPF_ANY);
    }



//    __u32 *pkt_saddr_count = bpf_map_lookup_elem(&tc_src_ip, &iph->saddr);
//    if (!pkt_saddr_count){
//        __u32 init_saddr_pkt_count = 1;
//        bpf_map_update_elem(&tc_src_ip, &iph->saddr, &init_saddr_pkt_count, BPF_ANY);
//    }else{
//        __sync_fetch_and_add(pkt_saddr_count, 1);
//    }
//
//    __u32 *pkt_daddr_count = bpf_map_lookup_elem(&tc_dst_ip, &iph->daddr);
//    if (!pkt_daddr_count){
//        __u32 init_daddr_pkt_count = 1;
//        bpf_map_update_elem(&tc_dst_ip, &iph->daddr, &init_daddr_pkt_count, BPF_ANY);
//    }else{
//        __sync_fetch_and_add(pkt_daddr_count, 1);
//    }

	return TC_ACT_OK;
}

