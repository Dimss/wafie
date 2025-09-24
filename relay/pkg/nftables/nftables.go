package nftables

import (
	"context"

	"sigs.k8s.io/knftables"
)

const (
	WafieGatewayNatTable        = "wafie-gateway"
	WafieGatewayPreroutingChain = "prerouting"
)

func Program(errChan chan error) {
	nft, err := knftables.New(knftables.InetFamily, WafieGatewayNatTable)
	if err != nil {
		errChan <- err
		return
	}

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
	errChan <- nft.Run(context.Background(), tx)

}
