package nftables

import (
	"context"
	"log"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"sigs.k8s.io/knftables"
)

const (
	WafieGatewayNatTable        = "wafie-gateway"
	WafieGatewayPreroutingChain = "prerouting"
)

var logger = log.New(os.Stderr, "[wafie-cni] ", log.LstdFlags)

func Program(netNsName string) error {
	netNs, err := ns.GetNS(netNsName)
	if err != nil {
		return err
	}
	defer netNs.Close()

	return netNs.Do(func(_ ns.NetNS) error {
		return applyNft()
	})
}

func startRelay() error {
	// start socat here
	// socat TCP-LISTEN:9090,fork TCP:10.96.109.104:8888
}

func applyNft() error {
	nft, err := knftables.New(knftables.InetFamily, WafieGatewayNatTable)
	if err != nil {
		return nil
	}
	chains, err := nft.List(context.Background(), "chains")
	if err != nil {
		return err
	}
	logger.Println("--------------- CHAINS ---------------")
	logger.Println(chains)
	logger.Println("--------------- END ---------------")

	table := knftables.Table{
		Family: knftables.InetFamily,
		Name:   WafieGatewayNatTable,
	}

	chain := knftables.Chain{
		Name:     WafieGatewayPreroutingChain,
		Table:    WafieGatewayNatTable,
		Family:   knftables.InetFamily,
		Type:     knftables.PtrTo(knftables.NATType),
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.DNATPriority),
	}

	tx := nft.NewTransaction()
	tx.Add(&table)
	tx.Add(&chain)

	ingressPodIp := "10.244.0.16"
	dstPort := "8080"

	rule := knftables.Rule{
		Table: WafieGatewayNatTable,
		Chain: WafieGatewayPreroutingChain,
		Rule: knftables.Concat(
			"ip saddr", ingressPodIp,
			"tcp dport", dstPort,
			"redirect to :9090",
		),
	}

	tx.Add(&rule)
	return nft.Run(context.Background(), tx)

}
