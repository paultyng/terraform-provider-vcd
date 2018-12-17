---
layout: "vcd"
page_title: "vCloudDirector: vcd_network"
sidebar_current: "docs-vcd-resource-network"
description: |-
  Provides a vCloud Director Org VDC Network. This can be used to create, modify, and delete internal networks for vApps to connect.
---

# vcd\_network (Deprecated)

Provides a vCloud Director Org VDC Network. This can be used to create,
modify, and delete internal networks for vApps to connect.

**Deprecated in v2.0+** : this resource is deprecated and replaced by [vcd-network-routed](vcd-network-routed).
It is also complemented by [vcd-network-isolated](vcd-network-isolated) and [vcd-network-direct](d-network-direct).

## Example Usage

```hcl
resource "vcd_network" "net" {
  name         = "my-net"
  edge_gateway = "Edge Gateway Name"
  gateway      = "10.10.0.1"

  dhcp_pool {
    start_address = "10.10.0.2"
    end_address   = "10.10.0.100"
  }

  static_ip_pool {
    start_address = "10.10.0.152"
    end_address   = "10.10.0.254"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the network
* `edge_gateway` - (Required) The name of the edge gateway
* `netmask` - (Optional) The netmask for the new network. Defaults to `255.255.255.0`
* `gateway` (Required) The gateway for this network
* `dns1` - (Optional) First DNS server to use. Defaults to `8.8.8.8`
* `dns2` - (Optional) Second DNS server to use. Defaults to `8.8.4.4`
* `dns_suffix` - (Optional) A FQDN for the virtual machines on this network
* `shared` - (Optional) Defines if this network is shared between multiple vDCs
  in the vOrg.  Defaults to `false`.
* `dhcp_pool` - (Optional) A range of IPs to issue to virtual machines that don't
  have a static IP; see [IP Pools](#ip-pools) below for details.
* `static_ip_pool` - (Optional) A range of IPs permitted to be used as static IPs for
  virtual machines; see [IP Pools](#ip-pools) below for details.

<a id="ip-pools"></a>
## IP Pools

Static IP Pools and DHCP Pools support the following attributes:

* `start_address` - (Required) The first address in the IP Range
* `end_address` - (Required) The final address in the IP Range

DHCP Pools additionally support the following attributes:

* `default_lease_time` - (Optional) The default DHCP lease time to use. Defaults to `3600`.
* `max_lease_time` - (Optional) The maximum DHCP lease time to use. Defaults to `7200`.
