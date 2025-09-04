//go:build ignore

#include "vmlinux.h"
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
