---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_external_network"
sidebar_current: "docs-vcd-data-source-external-network"
description: |-
  Provides an external network data source.
---

# external\_network

Provides a VMware Cloud Director external network data source. This can be used to reference external networks and their properties.

Supported in provider *v2.5+*

## Example Usage

```hcl
data "vcd_external_network" "tf-external-network" {
  name = "my-extnet"
}

resource "vcd_dnat" "tf-nat-rule" {
  org = "tf-org"
  vdc = "tf-vdc"
  # References the external network name from the data source
  network_name = data.vcd_external_network.tf-external-network.name
  network_type = "ext"
  edge_gateway = "tf-gw"
  # References the first IP scope block. From that we extract the first static IP pool to retrieve the start address
  external_ip     = "${data.vcd_external_network.extnet-datacloud.ip_scope[0].static_ip_pool[0].start_address}"
  port            = 7777
  protocol        = "tcp"
  internal_ip     = "10.10.102.60"
  translated_port = 77
  description     = "test run"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) external network name

## Attribute Reference

* `description` - Network friendly description
* `ip_scope` -  A list of IP scopes for the network. See [IP Scope](/docs/providers/vcd/r/external_network.html#ipscope)
   for details.
* `vsphere_network` -  A list of DV_PORTGROUP or NETWORK objects names that back this network. Each referenced 
  DV_PORTGROUP or NETWORK must exist on a vCenter server registered with the system.
  See [vSphere Network](/docs/providers/vcd/r/external_network.html#vspherenetwork) for details.
* `retain_net_info_across_deployments` -  Specifies whether the network resources such as IP/MAC of router will be 
  retained across deployments.

