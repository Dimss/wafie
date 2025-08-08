package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
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

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Please specify prog type [xdp|tc] and a network interface")
	}
	interfaces := getActiveInterfaces()

	// Look up the network interface by name.
	progType := os.Args[1]
	ifaceName := os.Args[2]
	if ifaceName == "all" {
		fmt.Println("using all interfaces")
	} else {
		_, err := net.InterfaceByName(ifaceName)
		if err != nil {
			log.Fatalf("lookup network iface %q: %s", ifaceName, err)
		}
		interfaces = []string{ifaceName}
	}

	// Load pre-compiled programs into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %s", err)
	}
	defer objs.Close()

	links, err := attachToMultipleInterfaces(progType, objs, interfaces)

	if err != nil {
		log.Fatalf("could not attach TCx program: %s", err)
	}
	defer func() {
		for _, l := range links {
			l.Close()
		}
	}()

	// Print the contents of the counters maps.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		readIPMap(objs.IpMap)
	}
}

func attachToMultipleInterfaces(progType string, obj bpfObjects, interfaceNames []string) ([]link.Link, error) {
	var links []link.Link

	for _, ifaceName := range interfaceNames {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("interface %s: %w", ifaceName, err)
		}
		if progType == "xdp" {
			l, err := link.AttachXDP(
				link.XDPOptions{
					Interface: iface.Index,
					Program:   obj.XdpProgFunc,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("attach to %s: %w", ifaceName, err)
			}
			fmt.Println("attaching to " + ifaceName)
			links = append(links, l)
		}
		if progType == "tc" {
			l, err := link.AttachTCX(link.TCXOptions{
				Interface: iface.Index,
				Program:   obj.IngressProgFunc,
				Attach:    ebpf.AttachTCXIngress,
			})
			if err != nil {
				return nil, fmt.Errorf("attach to %s: %w", ifaceName, err)
			}
			fmt.Println("attaching to " + ifaceName)
			links = append(links, l)
		}

	}
	return links, nil
}

func getActiveInterfaces() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var names []string
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		//if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
		names = append(names, iface.Name)
		//}
	}
	return names
}

func readIPMap(ipMap *ebpf.Map) {
	var key IpPair
	var value IpPairValue
	iter := ipMap.Iterate()

	for iter.Next(&key, &value) {
		srcIP := intToIP(key.SrcIP)
		dstIP := intToIP(key.DstIP)
		//wpSrcIp := net.ParseIP("10.244.0.7")
		//wpDstIp := net.ParseIP("10.244.0.10")
		//if srcIP.Equal(wpSrcIp) && dstIP.Equal(wpDstIp) {
		fmt.Printf("%s \t--------> %s: %d packets interface: %s\n",
			srcIP, dstIP, value.Count, ifindexToName(value.Ifindex))
		//}
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
