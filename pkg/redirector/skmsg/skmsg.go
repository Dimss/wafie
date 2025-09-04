package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/spf13/cobra"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux bpf sk_msg.c -- -I/usr/include

func init() {
	//rootCmd.PersistentFlags().StringP("iface-name", "i", "", "Interface name")
	//rootCmd.MarkPersistentFlagRequired("iface-name")
	//
	//viper.BindPFlag("iface-name", rootCmd.PersistentFlags().Lookup("iface-name"))
}

var rootCmd = &cobra.Command{
	Use:   "start tc ebpf program",
	Short: "start tc ebpf program",
	Run: func(cmd *cobra.Command, args []string) {
		objs := bpfObjects{}
		if err := loadBpfObjects(&objs, nil); err != nil {
			log.Fatalf("loading objects: %s", err)
		}
		defer objs.Close()
		attachProgram(objs)
	},
}

func attachProgram(objs bpfObjects) {
	sockMap, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.SockMap,
		KeySize:    4,
		ValueSize:  8,
		MaxEntries: 100,
	})
	if err != nil {
		log.Fatal("Create sockmap failed:", err)
	}
	defer sockMap.Close()
	l, err := link.AttachRawLink(link.RawLinkOptions{
		Target:  sockMap.FD(),
		Program: objs.CrossNamespaceRedirect,
		Attach:  ebpf.AttachSkMsgVerdict,
	})
	if err != nil {
		log.Fatal("Attach failed:", err)
	}
	defer l.Close()
	log.Println("SK_MSG program attached successfully!")
	log.Println("Press Ctrl+C to detach...")
	// Wait for signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	log.Println("Detaching program...")
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
