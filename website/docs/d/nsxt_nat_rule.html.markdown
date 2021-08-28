---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_nat_rule"
sidebar_current: "docs-vcd-data-source-nsxt-nat-rule"
description: |-
  Provides a data source to read NSX-T NAT rules. Source NAT (SNAT) rules change the source IP 
  address from a private to a public IP address. Destination NAT (DNAT) rules change the destination
  IP address from a public to a private IP address.
---

# vcd\_nsxt\_nat\_rule

Supported in provider *v3.3+* and VCD 10.1+ with NSX-T backed VDCs.

Provides a data source to read NSX-T NAT rules. Source NAT (SNAT) rules change the source IP 
address from a private to a public IP address. Destination NAT (DNAT) rules change the destination
IP address from a public to a private IP address.

## Example Usage

```hcl
data "vcd_nsxt_nat_rule" "dnat-ssh" {
  org = "my-org"
  vdc = "my-org-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name = "dnat-ssh"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organizations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway_id` - (Required) The ID of the Edge Gateway (NSX-T only). Can be looked up using
* `name` - (Required)  - Name of existing NAT Rule.

-> Name uniqueness is not enforced in NSX-T NAT rules, but for this data source to work properly
names should be unique so that they can be distinguished.

## Attribute Reference

All the arguments and attributes defined in
[`vcd_nsxt_nat_rule`](/docs/providers/vcd/r/nsxt_nat_rule.html) resource are available.
