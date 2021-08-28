---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_network_isolated"
sidebar_current: "docs-vcd-data-source-network-isolated"
description: |-
  Provides a VMware Cloud Director Org VDC isolated Network. This can be used to reference internal networks for vApps to connect.
---

# vcd\_network\_isolated

Provides a VMware Cloud Director Org VDC isolated Network data source. This can be used to reference
internal networks for vApps to connect. This network is not attached to external networks or routers.

Supported in provider *v2.5+*

~> **Note:** This data source supports only NSX-V backed Org VDC networks.
Please use newer [`vcd_network_isolated_v2`](/docs/providers/vcd/d/network_isolated_v2.html)
data source which is compatible with NSX-T.

## Example Usage

```hcl
data "vcd_network_isolated" "net" {
  org = "my-org" # Optional
  vdc = "my-vdc" # Optional

  name = "my-net"
}

output "gateway" {
  value = data.vcd_network_isolated.net.gateway
}

output "dns1" {
  value = data.vcd_network_isolated.net.dns1
}

output "dhcp_start_address" {
  value = tolist(data.vcd_network_isolated.net.dhcp_pool)[0].start_address
}

output "dhcp_end_address" {
  value = tolist(data.vcd_network_isolated.net.dhcp_pool)[0].end_address
}

output "static_ip_start_address" {
  value = tolist(data.vcd_network_isolated.net.static_ip_pool)[0].start_address
}

output "static_ip_end_address" {
  value = tolist(data.vcd_network_isolated.net.static_ip_pool)[0].end_address
}

```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `name` - (Required) A unique name for the network (optional when `filter` is used)
* `filter` - (Optional; *2.9+*) Retrieves the data source using one or more filter parameters

## Attribute reference

All attributes defined in [isolated network resource](/docs/providers/vcd/r/network_isolated.html#attribute-reference) are supported.

## Filter arguments

(Supported in provider *v2.9+*)

* `name_regex` (Optional) matches the name using a regular expression.
* `ip` (Optional) matches the IP of the resource using a regular expression.
* `metadata` (Optional) One or more parameters that will match metadata contents.

See [Filters reference](/docs/providers/vcd/guides/data_source_filters.html) for details and examples.
