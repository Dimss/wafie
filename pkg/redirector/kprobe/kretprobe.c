//go:build ignore

#include "vmlinux.h"
//#include <linux/in.h>
//#include <linux/socket.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct sock_args {
    int family;
    int type;
    int protocol;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, u64);
    __type(value, struct sock_args);
} socket_args SEC(".maps");


// Data stored temporarily between the two probes
struct net_info {
    u32 saddr;
    u32 daddr;
    u16 sport;
    u16 dport;
};

// Map to correlate data between probes, keyed by thread ID
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10240);
    __type(key, u64);
    __type(value, struct net_info);
} active_accepts SEC(".maps");

// Perf buffer to send events to userspace
struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} events SEC(".maps");




// 1. This probe fires first when the kernel TCP stack accepts a connection.
// It has the network details but not the file descriptor.
SEC("kretprobe/inet_csk_accept")
int kretprobe_inet_csk_accept(struct pt_regs *ctx) {

    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u64 my_cgroup_id = 27917;
    if (cgroup_id != my_cgroup_id) {
            return 0;
    }

    struct sock *new_sk = (struct sock *)PT_REGS_RC(ctx);

    if (!new_sk) {
        return 0;
    }

    unsigned short family;
    bpf_probe_read_kernel(&family, sizeof(family), &new_sk->__sk_common.skc_family);

    bpf_printk("Accepted socket family: %d", family);
    if (family != AF_INET) {
        return 0;
    }

    bpf_printk("SRC IPv4: %pI4\n", &new_sk->__sk_comm


    );
//    if (!new_sk) {
//        return 0;
//    }
//
//    __u64 cgroup_id = bpf_get_current_cgroup_id();
//    __u64 my_cgroup_id = 27917;
//    if (cgroup_id != my_cgroup_id) {
//            return 0;
//    }
//
//    struct net_info info = {};
//
//    // Use bpf_probe_read to safely access kernel memory
//    struct sock_common skc;
//    if (bpf_probe_read(&skc, sizeof(skc), &new_sk->__sk_common) != 0) {
//        return 0;
//    }
//
//    info.saddr = skc.skc_rcv_saddr;
//    info.daddr = skc.skc_daddr;
//    info.sport = skc.skc_num;
//    info.dport = __builtin_bswap16(skc.skc_dport); // Convert from network byte order
//
//    // Fix bpf_printk format - remove extra parameters
//    bpf_printk("SRC IPv4: %pI4:%d\n", &info.saddr, info.sport);
//    bpf_printk("DST IPv4: %pI4:%d\n", &info.daddr, info.dport);
    // Store the details, keyed by the current thread ID
//    bpf_map_update_elem(&active_accepts, &id, &info, BPF_ANY);
    return 0;
}

SEC("kprobe/__x64_sys_socket")
int kprobe_sys_socket(struct pt_regs *ctx) {
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u64 my_cgroup_id = 27917;
    if (cgroup_id != my_cgroup_id) {
        return 0;
    }
    __u64 id = bpf_get_current_pid_tgid();
    struct sock_args args = {};
    args.family   = PT_REGS_PARM1(ctx);
    args.type     = PT_REGS_PARM2(ctx);
    args.protocol = PT_REGS_PARM3(ctx);
//    bpf_printk("SOCK FAMILY: %lu\n", args.family);
//    bpf_printk("SOCK TYPE: %lu\n", args.type);
//    bpf_printk("SOCK PROTOCOL: %lu\n", args.protocol);
    bpf_map_update_elem(&socket_args, &id, &args, BPF_ANY);
    return 0;
}

SEC("kretprobe/__x64_sys_socket")
int kretprobe_sys_socket(struct pt_regs *ctx) {
    __u64 id = bpf_get_current_pid_tgid();
    struct sock_args *args;

    args = bpf_map_lookup_elem(&socket_args, &id);
    if (!args) {
        // Should not happen if the kprobe ran.
        return 0;
    }

    // Get the return value from the syscall.
    int ret_fd = PT_REGS_RC(ctx);

    // Clean up the map entry. This is crucial!
    bpf_map_delete_elem(&socket_args, &id);

    // We only care about successfully created sockets.
    if (ret_fd < 0) {
        return 0;
    }

    bpf_printk("SOCKET FD: %d\n", ret_fd);

    return 0;
}

SEC("kprobe/__x64_sys_connect")
int kprobe_sys_connect(struct pt_regs *ctx) {
    __u64 cgroup_id = bpf_get_current_cgroup_id();

    __u64 my_cgroup_id = 27917;
    if (cgroup_id != my_cgroup_id) {
        return 0;
    }

    int sockfd = PT_REGS_PARM1(ctx);
    struct sockaddr *addr = (struct sockaddr *)PT_REGS_PARM2(ctx);
    struct sockaddr sa;
    if (bpf_probe_read(&sa, sizeof(sa), addr) != 0) {
        return 0;
    }
    bpf_printk("AF_INET: %d\n", sa.sa_family);
    if (sa.sa_family != 2) { // AF_INET = 2
        return 0;
    }

    struct sockaddr_in addr_in;
    if (bpf_probe_read(&addr_in, sizeof(addr_in), addr) != 0) {
        return 0;
    }

    __u32 ip = addr_in.sin_addr.s_addr;
    __u16 port = __builtin_bswap16(addr_in.sin_port);
    bpf_printk("Connect FD:%d to %pI4:%d\n", sockfd, &ip, port);
    return 0;
  }
