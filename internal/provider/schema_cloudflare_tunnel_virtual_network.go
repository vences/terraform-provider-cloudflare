package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareTunnelVirtualNetworkSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"account_id": {
			Description: "The account identifier to target for the resource.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		"name": {
			Description: "A user-friendly name chosen when the virtual network is created.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"is_default_network": {
			Description: "Whether this virtual network is the default one for the account. This means IP Routes belong to this virtual network and Teams Clients in the account route through this virtual network, unless specified otherwise for each case.",
			Type:        schema.TypeBool,
			Optional:    true,
		},
		"comment": {
			Description: "Description of the tunnel virtual network.",
			Type:        schema.TypeString,
			Optional:    true,
		},
	}
}
