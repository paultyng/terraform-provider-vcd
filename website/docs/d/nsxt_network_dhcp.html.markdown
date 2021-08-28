---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_network_dhcp"
sidebar_current: "docs-vcd-datasource-nsxt-network-dhcp"
description: |-
  Provides a data source to read DHCP pools for NSX-T Org VDC Routed network.
---

# vcd\_nsxt\_network\_dhcp

Provides a data source to read DHCP pools for NSX-T Org VDC Routed network.

Supported in provider *v3.2+* and VCD 10.1+ with NSX-T backed VDCs.

## Example Usage 1

```hcl

data "vcd_network_routed_v2" "parent" {
  org = "my-org"
  vdc = "my-vdc"

  name = "my-parent-network"
}

data "vcd_nsxt_network_dhcp" "pools" {
  org = "my-org"
  vdc = "my-vdc"

  org_network_id = vcd_network_routed_v2.parent.id
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `org_network_id` - (Required) ID of parent Org VDC Routed network

## Attribute Reference

All the attributes defined in [`vcd_nsxt_network_dhcp`](/docs/providers/vcd/r/nsxt_network_dhcp.html)
resource are available.
