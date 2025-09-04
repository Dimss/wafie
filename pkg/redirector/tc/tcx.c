//go:build ignore
#include <linux/tcp.h>
#include "bpf_endian.h"
#include "common.h"

char __license[] SEC("license") = "Dual MIT/GPL";

#define MAX_MAP_ENTRIES 128

#define IP_ADDRESS(a, b, c, d) \
  ((__be32)(((__u32)(a) << 24) | ((__u32)(b) << 16) | \
           ((__u32)(c) << 8) | (__u32)(d)))


//const volatile __be32 proxy_ip = 0;

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

SEC("xdp")
int redirect_to_proxy_xdp(struct xdp_md *ctx) {
	void *data_end = (void *)(long)ctx->data_end;
	void *data     = (void *)(long)ctx->data;

    // Pointers for parsing
    struct ethhdr *eth = data;
    struct iphdr *iph;

  // Boundary check and ensure it's an IP packet
    if (data + sizeof(*eth) > data_end)
        return XDP_PASS;
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    iph = data + sizeof(*eth);
    if ((void *)iph + sizeof(*iph) > data_end)
        return XDP_PASS;

    if (iph->protocol != IPPROTO_TCP)
        return XDP_PASS;

    struct tcphdr *tcph = (void *)iph + sizeof(*iph);
    if ((void *)tcph + sizeof(*tcph) > data_end) {
        return XDP_PASS; // Not enough space for TCP header
    }

    __u16 dest_port = bpf_ntohs(tcph->dest);
    __u16 source_port = bpf_ntohs(tcph->source);

    bpf_printk("DESTINATION PORT: %d\n", dest_port);
    bpf_printk("SOURCe PORT: %d\n", source_port);

    struct ip_pair key = {
        .src_ip = iph->saddr,
        .dst_ip = iph->daddr,
    };
    __u32 src_ip = __builtin_bswap32(iph->saddr);
    struct ip_pair_value *ipv = bpf_map_lookup_elem(&ip_map, &key);
    if (ipv) {
        __sync_fetch_and_add(&ipv->count, 1);
//         bpf_printk("SRC: %u.%u.%u.%u",
//                    (src_ip >> 24) & 0xFF,
//                    (src_ip >> 16) & 0xFF,
//                    (src_ip >> 8) & 0xFF,
//                     src_ip & 0xFF);
    } else {
//        bpf_printk("iface name: %d", ctx->ifindex);
//        bpf_printk("SRC: %u.%u.%u.%u",
//                   (src_ip >> 24) & 0xFF,
//                   (src_ip >> 16) & 0xFF,
//                   (src_ip >> 8) & 0xFF,
//                    src_ip & 0xFF);
        struct ip_pair_value value = {
            .count = 1,
            .ifindex = ctx->ingress_ifindex
        };
        bpf_map_update_elem(&ip_map, &key, &value, BPF_ANY);
    }
//     __be32 source_ip = bpf_htonl(0x0AF4001F);
//     bpf_printk("Expected IP: %pI4 (0x%08x)\n", &source_ip, source_ip);
//     bpf_printk("Actual IP: %pI4 (0x%08x)\n", &iph->saddr, iph->saddr);
//     bpf_printk("Are equal? %s\n", (iph->saddr == source_ip) ? "YES" : "NO");
//     bpf_printk("###########################################################");
//    if (iph->saddr == source_ip){
//        bpf_printk("SOURCE in header and defined IPs are equals!!");
//        // Target IP you want to redirect to
//        __be32 proxy_ip = bpf_htonl(0x0AF40032);
//        // Modify destination IP
//        __u32 old_daddr = iph->daddr;
//        iph->daddr = proxy_ip;
//        bpf_printk("Redirecting to: %pI4 (0x%08x)\n", &iph->daddr, iph->daddr);
//        // Update IP checksum
//        __u32 csum = bpf_csum_diff(&old_daddr, sizeof(old_daddr),
//                                   &proxy_ip, sizeof(proxy_ip), 0);
//        bpf_l3_csum_replace(skb, offsetof(struct iphdr, check), 0, csum, 0);
//        // Redirect to appropriate interface
//        return bpf_redirect(skb->ifindex, 0);
//    }
	return XDP_PASS;
}


SEC("tc")
int redirect_to_proxy(struct __sk_buff *skb) {
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

    struct tcphdr *tcph = (void *)iph + sizeof(*iph);
    if ((void *)tcph + sizeof(*tcph) > data_end) {
        return TC_ACT_OK; // Not enough space for TCP header
    }

    __u16 dest_port = bpf_ntohs(tcph->dest);
    __u16 source_port = bpf_ntohs(tcph->source);



//    bpf_printk("DESTINATION PORT: %d\n", dest_port);
//    bpf_printk("SOURCE PORT: %d\n", source_port);
    if (dest_port != 8080) return TC_ACT_OK;
    bpf_printk("%pI4:%d ----> %pI4:%d \n", &iph->saddr, source_port, &iph->daddr, dest_port);


    struct ip_pair key = {
        .src_ip = iph->saddr,
        .dst_ip = iph->daddr,
    };
    __u32 src_ip = __builtin_bswap32(iph->saddr);
    struct ip_pair_value *ipv = bpf_map_lookup_elem(&ip_map, &key);
    if (ipv) {
        __sync_fetch_and_add(&ipv->count, 1);
//         bpf_printk("SRC: %u.%u.%u.%u",
//                    (src_ip >> 24) & 0xFF,
//                    (src_ip >> 16) & 0xFF,
//                    (src_ip >> 8) & 0xFF,
//                     src_ip & 0xFF);
    } else {
//        bpf_printk("iface name: %d", skb->ifindex);
//        bpf_printk("SRC: %u.%u.%u.%u",
//                   (src_ip >> 24) & 0xFF,
//                   (src_ip >> 16) & 0xFF,
//                   (src_ip >> 8) & 0xFF,
//                    src_ip & 0xFF);
        struct ip_pair_value value = {
            .count = 1,
            .ifindex = skb->ifindex
        };
        bpf_map_update_elem(&ip_map, &key, &value, BPF_ANY);
    }
//     __be32 source_ip = bpf_htonl(0x0AF4001F);
//     bpf_printk("Expected IP: %pI4 (0x%08x)\n", &source_ip, source_ip);
//     bpf_printk("Actual IP: %pI4 (0x%08x)\n", &iph->saddr, iph->saddr);
//     bpf_printk("Are equal? %s\n", (iph->saddr == source_ip) ? "YES" : "NO");
//     bpf_printk("###########################################################");
//    if (iph->saddr == source_ip){
//        bpf_printk("SOURCE in header and defined IPs are equals!!");
//        // Target IP you want to redirect to
//        __be32 proxy_ip = bpf_htonl(0x0AF40032);
//        // Modify destination IP
//        __u32 old_daddr = iph->daddr;
//        iph->daddr = proxy_ip;
//        bpf_printk("Redirecting to: %pI4 (0x%08x)\n", &iph->daddr, iph->daddr);
//        // Update IP checksum
//        __u32 csum = bpf_csum_diff(&old_daddr, sizeof(old_daddr),
//                                   &proxy_ip, sizeof(proxy_ip), 0);
//        bpf_l3_csum_replace(skb, offsetof(struct iphdr, check), 0, csum, 0);
//        // Redirect to appropriate interface
//        return bpf_redirect(skb->ifindex, 0);
//    }
	return TC_ACT_OK;
}