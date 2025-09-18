package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf/link"
	"github.com/spf13/cobra"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux bpf kretprobe.c -- -I/usr/include -I./headers -target bpf -D__TARGET_ARCH_arm64

func init() {

}

var rootCmd = &cobra.Command{
	Use:   "start kretprobe ebpf program",
	Short: "start kretprobe ebpf program",
	Run: func(cmd *cobra.Command, args []string) {
		objs := bpfObjects{}
		if err := loadBpfObjects(&objs, nil); err != nil {
			log.Fatalf("loading objects: %v", err)
		}
		defer objs.Close()

		// Attach the kprobe.
		//kp, err := link.Kprobe("__arm64_sys_socket", objs.KprobeSysSocket, nil)
		//if err != nil {
		//	log.Fatalf("attaching kprobe: %s", err)
		//}
		//defer kp.Close()
		//
		//Attach the kretprobe.
		krp, err := link.Kretprobe("__arm64_sys_socket", objs.KretprobeInetCskAccept, nil)
		if err != nil {
			log.Fatalf("attaching kretprobe: %s", err)
		}
		defer krp.Close()

		// Attach kprobe/__x64_sys_connect
		//log.Println("attaching __arm64_sys_connect")
		//kpSysConnect, err := link.Kprobe("__arm64_sys_connect", objs.KprobeSysConnect, nil)
		//if err != nil {
		//	log.Fatalf("attaching kprobe: %s", err)
		//}
		//defer kpSysConnect.Close()

		// Wait for signal
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		log.Println("Detaching program...")
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
