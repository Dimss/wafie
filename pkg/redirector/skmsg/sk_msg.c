//go:build ignore
#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

char __license[] SEC("license") = "Dual MIT/GPL";

SEC("sk_msg")
int cross_namespace_redirect(struct sk_msg_md *msg) {
    bpf_printk("LOCAL_IP4: %pI4\n", msg->local_ip4);
    return SK_PASS;
}