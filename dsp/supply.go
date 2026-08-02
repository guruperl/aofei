package dsp

import (
	"fmt"
	"strings"

	"github.com/guruperl/aofei/acl"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func sourceFromApprovedSeller(seller acl.SellerMetadata) *openrtb2.Source {
	if !seller.Authorized || seller.Validate() != nil {
		return nil
	}
	hop := int8(1)
	complete := int8(1)
	if seller.Type == "Intermediary" {
		// W8M knows the approved reseller account but cannot claim that an
		// unrecorded upstream owner is present in the chain.
		complete = 0
	}
	return &openrtb2.Source{SChain: &openrtb2.SupplyChain{
		Ver:      "1.0",
		Complete: complete,
		Nodes: []openrtb2.SupplyChainNode{{
			ASI: seller.ASI, SID: seller.ID, HP: &hop,
		}},
	}}
}

// validatePartnerSource validates standard schain fields after the S01
// sanitizer has removed extensions. It preserves a valid partner chain and
// rejects malformed or overlong input instead of forwarding raw claims.
func validatePartnerSource(source *openrtb2.Source) error {
	if source == nil || source.SChain == nil {
		return nil
	}
	chain := source.SChain
	if chain.Ver != "1.0" || (chain.Complete != 0 && chain.Complete != 1) || len(chain.Nodes) == 0 || len(chain.Nodes) > 10 {
		return fmt.Errorf("middleman source.schain has an invalid envelope")
	}
	for index := range chain.Nodes {
		node := &chain.Nodes[index]
		if len(node.RID) > 128 || node.RID != strings.TrimSpace(node.RID) || node.Ext != nil {
			return fmt.Errorf("middleman source.schain node %d has invalid optional fields", index)
		}
		if node.HP == nil || *node.HP != 1 {
			return fmt.Errorf("middleman source.schain node %d has invalid hp", index)
		}
		if err := (acl.SellerMetadata{
			ID: node.SID, Type: "Publisher", ASI: node.ASI,
			Name: node.Name, Domain: node.Domain, Authorized: true,
		}).Validate(); err != nil {
			return fmt.Errorf("middleman source.schain node %d is invalid: %w", index, err)
		}
	}
	if chain.Ext != nil {
		return fmt.Errorf("middleman source.schain extensions are not approved")
	}
	return nil
}
