package vcd

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func VAppNetworkSubresourceSchema() map[string]*schema.Schema {
	// VALIDATE IPS!!!!
	s := map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},

		"gateway": {
			Type:     schema.TypeString,
			Required: true,
		},
		"netmask": {
			Type:     schema.TypeString,
			Required: true,
		},
		"dns1": {
			Type:     schema.TypeString,
			Required: true,
		},
		"dns2": {
			Type:     schema.TypeString,
			Required: true,
		},
		"start": {
			Type:     schema.TypeString,
			Required: true,
		},
		"end": {
			Type:     schema.TypeString,
			Required: true,
		},

		"nat": {
			Type:     schema.TypeBool,
			Required: true,
		},
		"parent": {
			Type:     schema.TypeString,
			Required: true,
		},
		"dhcp": {
			Type:     schema.TypeBool,
			Required: true,
		},
	}
	return s
}

type VAppNetworkSubresource struct {
	*Subresource
}

func NewVAppNetworkSubresource(d, old map[string]interface{}) *VAppNetworkSubresource {
	sr := &VAppNetworkSubresource{
		Subresource: &Subresource{
			schema:  VAppNetworkSubresourceSchema(),
			data:    d,
			olddata: old,
			// rdd:     rdd,
		},
	}
	// sr.Index = idx
	return sr
}

// Suppress Diff on equal ip
// func suppressIPDifferences(k, old, new string, d *schema.ResourceData) bool {
// 	o := net.ParseIP(old)
// 	n := net.ParseIP(new)

// 	if o != nil && n != nil {
// 		return o.Equal(n)
// 	}
// 	return false
// }
