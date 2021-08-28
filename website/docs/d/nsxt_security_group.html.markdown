---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_security_group"
sidebar_current: "docs-vcd-data-source-nsxt-security-group"
description: |-
  Provides a data source to access NSX-T Security Group configuration. Security groups are groups of
  data center group networks to which distributed firewall rules apply. Grouping networks helps you 
  to reduce the total number of distributed firewall rules to be created. 
---

# vcd\_nsxt\_security\_group

Supported in provider *v3.3+* and VCD 10.1+ with NSX-T backed VDCs.

Provides a data source to access NSX-T Security Group configuration. Security groups are groups of
data center group networks to which distributed firewall rules apply. Grouping networks helps you to
reduce the total number of distributed firewall rules to be created.

## Example Usage 1

```hcl
data "vcd_nsxt_security_group" "group1" {
  org = "my-org"
  vdc = "my-org-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name = "test-security-group-changed"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway_id` - (Required) The ID of the edge gateway (NSX-T only). Can be looked up using
* `name` - (Required)  - Unique name of existing Security Group.

## Attribute Reference

All the arguments and attributes defined in
[`vcd_nsxt_security_group`](/docs/providers/vcd/r/nsxt_security_group.html) resource are available.
