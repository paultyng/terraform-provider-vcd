---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxv_snat"
sidebar_current: "docs-vcd-data-source-nsxv-snat"
description: |-
  Provides a VMware Cloud Director SNAT data source for advanced edge gateways (NSX-V). This can be used to read
  existing rule by ID and use its attributes in other resources.
---

# vcd\_nsxv\_snat

Provides a VMware Cloud Director SNAT data source for advanced edge gateways (NSX-V). This can be used to
read existing rule by ID and use its attributes in other resources.

~> **Note:** This data source requires advanced edge gateway.

## Example Usage

```hcl
data "vcd_nsxv_snat" "my-rule" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  rule_id = "197867"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway` - (Required) The name of the edge gateway on which to apply the SNAT rule.
* `rule_id` - (Required) ID of SNAT rule as shown in the UI.

## Attribute Reference

All the attributes defined in `vcd_nsxv_snat` resource are available.
