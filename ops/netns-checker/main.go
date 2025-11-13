//go:build linux

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/containernetworking/plugins/pkg/ns"
)

func main() {
	var netns = flag.String("netns", "", "netns path")

	flag.Parse()
	if netns == nil || *netns == "" {
		log.Println(" netns must be set")
		os.Exit(1)
	}
	if err := runInNs(*netns); err != nil {
		log.Println(err)
	}
}

func runInNs(netnsPath string) error {
	var netNs ns.NetNS
	defer func(netNs ns.NetNS) {
		if netNs != nil {
			netNs.Close()
		}
	}(netNs)
	netNs, err := ns.GetNS(netnsPath)
	if err != nil {
		return err
	}
	return netNs.Do(func(_ ns.NetNS) error {

		cmd := exec.Command("ip", "address")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		output, err := cmd.Output()
		log.Println(string(output))
		return err
	})
}
