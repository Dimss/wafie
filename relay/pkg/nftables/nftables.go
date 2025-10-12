package nftables

import (
	"context"

	"sigs.k8s.io/knftables"
)

type (
	operation string
)

const (
	WafieGatewayNatTable                  = "wafie-gateway"
	WafieGatewayPreroutingChain           = "prerouting"
	AddOp                       operation = "add"
	DeleteOp                    operation = "delete"
)

func Program(op operation) error {
	nft, err := knftables.New(knftables.InetFamily, WafieGatewayNatTable)
	if err != nil {
		return err
	}
	tx := nft.NewTransaction()
	if op == AddOp {
		add(tx)
	}
	if op == DeleteOp {
		delete(tx)
	}
	return nft.Run(context.Background(), tx)
}

func add(tx *knftables.Transaction) {
	tx.Add(table())
	tx.Add(chain())
	tx.Add(rule())
}

func delete(tx *knftables.Transaction) {
	//tx.Delete(rule())
	//tx.Delete(chain())
	tx.Delete(table())
}

func table() *knftables.Table {
	return &knftables.Table{
		Family: knftables.InetFamily,
		Name:   WafieGatewayNatTable,
	}
}

func chain() *knftables.Chain {
	return &knftables.Chain{
		Name:     WafieGatewayPreroutingChain,
		Table:    WafieGatewayNatTable,
		Family:   knftables.InetFamily,
		Type:     knftables.PtrTo(knftables.NATType),
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.DNATPriority),
	}
}

func rule() *knftables.Rule {
	ingressPodIp := "10.244.0.7"
	dstPort := "8080"
	return &knftables.Rule{
		Table: WafieGatewayNatTable,
		Chain: WafieGatewayPreroutingChain,
		Rule: knftables.Concat(
			"ip saddr", ingressPodIp,
			"tcp dport", dstPort,
			"redirect to :9090",
		),
	}
}
