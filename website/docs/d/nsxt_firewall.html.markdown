---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_firewall"
sidebar_current: "docs-vcd-resource-nsxt-firewall"
description: |-
  Provides a data source to read NSX-T Firewall configuration of an Edge Gateway. Firewalls allow 
  user to control the incoming and outgoing network traffic to and from an NSX-T Data Center 
  Edge Gateway.
---

# vcd\_nsxt\_firewall

Supported in provider *v3.3+* and VCD 10.1+ with NSX-T backed Edge Gateways.

Provides a data source to read NSX-T Firewall configuration of an Edge Gateway. Firewalls allow 
user to control the incoming and outgoing network traffic to and from an NSX-T Data Center 
Edge Gateway.


## Example Usage (Read a list of firewall rules on Edge Gateway)
```hcl
data "vcd_nsxt_firewall" "testing" {
  org = "my-org"
  vdc = "my-nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.testing.id
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organizations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway_id` - (Required) The ID of the Edge Gateway (NSX-T only). Can be looked up using
  `vcd_nsxt_edgegateway` data source

## Attribute reference

All properties defined in [vcd_nsxt_firewall](/docs/providers/vcd/r/nsxt_firewall.html)
resource are available.
