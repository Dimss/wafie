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
	WafieOwnedComment                     = "wafie-owned-object"
	AddOp                       operation = "add"
	DeleteOp                    operation = "delete"
)

func Program(op operation) error {
	nft, err := knftables.New(knftables.InetFamily, WafieGatewayNatTable)
	if err != nil {
		return err
	}
	tx := nft.NewTransaction()
	// create nft rules
	if op == AddOp {
		rulesApplied, err := rulesState(nft)
		if err != nil {
			return err
		}
		if !rulesApplied {
			add(tx)
		}
	}
	// delete nft rules
	if op == DeleteOp {
		rulesApplied, err := rulesState(nft)
		if err != nil {
			return err
		}
		if rulesApplied {
			delete(tx)
		}
	}
	return nft.Run(context.Background(), tx)
}

func rulesState(nft knftables.Interface) (applied bool, err error) {
	chains, err := nft.List(context.Background(), "chains")
	// in case of error, do not program anything
	if err != nil {
		return true, err
	}
	// if no chains exists, program is required
	if len(chains) == 0 {
		return false, nil
	}
	// if chains list includes WafieGatewayPreroutingChain, further checks are required
	for _, chain := range chains {
		// first make sure the chain with the WafieGatewayPreroutingChain name exists
		if chain == WafieGatewayPreroutingChain {
			// list all rules in the WafieGatewayPreroutingChain chain
			rules, err := nft.ListRules(context.Background(), WafieGatewayPreroutingChain)
			// in case of error, do not program anything
			if err != nil {
				return true, err
			}
			// if no rules are found in the chain, program is required
			if len(rules) == 0 {
				return false, nil
			}
			// make sure the chain have at least one rule with WafieOwnedComment comment
			for _, rule := range rules {
				if *rule.Comment == WafieOwnedComment {
					return true, nil
				}
			}
			return false, nil
		}
	}
	// program required
	return false, nil

}

func add(tx *knftables.Transaction) {
	tx.Add(table())
	tx.Add(chain())
	tx.Add(rule())
}

func delete(tx *knftables.Transaction) {
	tx.Delete(table())
}

func table() *knftables.Table {
	return &knftables.Table{
		Family: knftables.InetFamily,
		Name:   WafieGatewayNatTable,
	}
}

func chain() *knftables.Chain {
	comment := WafieOwnedComment
	return &knftables.Chain{
		Name:     WafieGatewayPreroutingChain,
		Table:    WafieGatewayNatTable,
		Family:   knftables.InetFamily,
		Type:     knftables.PtrTo(knftables.NATType),
		Hook:     knftables.PtrTo(knftables.PreroutingHook),
		Priority: knftables.PtrTo(knftables.DNATPriority),
		Comment:  &comment,
	}
}

func rule() *knftables.Rule {
	ingressPodIp := "10.244.0.7"
	dstPort := "8080"
	comment := WafieOwnedComment
	return &knftables.Rule{
		Table:   WafieGatewayNatTable,
		Chain:   WafieGatewayPreroutingChain,
		Comment: &comment,
		Rule: knftables.Concat(
			"ip saddr", ingressPodIp,
			"tcp dport", dstPort,
			"redirect to :9090",
		),
	}
}
