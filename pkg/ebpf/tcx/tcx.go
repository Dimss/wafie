package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux bpf tcx.c -- -I../headers

type IpPair struct {
	SrcIP uint32 // __be32 in network byte order
	DstIP uint32 // __be32 in network byte order
}

type IpPairValue struct {
	Count   uint64 // __u64 to match C struct
	Ifindex uint32
	_       uint32
}

func init() {
	rootCmd.PersistentFlags().String("iface-name", "i", "Interface name")

	viper.BindPFlag("iface-name", rootCmd.PersistentFlags().Lookup("iface-name"))
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
		i := interfaceLookup(viper.GetString("iface-name"))
		link, err := attachTCProgram(i, objs)
		defer link.Close()
		if err != nil {
			log.Fatal(err)
		}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			readIPMap(objs.IpMap)
		}
	},
}

func attachTCProgram(i *net.Interface, objs bpfObjects) (link.Link, error) {
	return link.AttachTCX(
		link.TCXOptions{
			Interface: i.Index,
			Program:   objs.IngressProgFunc,
			Attach:    ebpf.AttachTCXIngress,
		},
	)
}

func interfaceLookup(ifname string) *net.Interface {
	i, err := net.InterfaceByName(ifname)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

func readIPMap(ipMap *ebpf.Map) {
	var key IpPair
	var value IpPairValue
	iter := ipMap.Iterate()

	for iter.Next(&key, &value) {
		srcIP := intToIP(key.SrcIP)
		dstIP := intToIP(key.DstIP)
		wpSrcIp := net.ParseIP("10.244.0.31")
		wpDstIp := net.ParseIP("10.244.0.46")
		if srcIP.Equal(wpSrcIp) && dstIP.Equal(wpDstIp) {
			fmt.Printf("%s \t--------> %s: %d packets interface: %s\n",
				srcIP, dstIP, value.Count, ifindexToName(value.Ifindex))
		}
	}

	if err := iter.Err(); err != nil {
		log.Printf("Iterator error: %v", err)
	}
}

func intToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
}

func ifindexToName(ifindex uint32) string {
	iface, err := net.InterfaceByIndex(int(ifindex))
	if err != nil {
		return fmt.Sprintf("unknown(%d)", ifindex)
	}
	return iface.Name
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
