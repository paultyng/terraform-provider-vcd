---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vm_sizing_policy"
sidebar_current: "docs-vcd-datasource-vm-sizing-policy"
description: |-
  Provides a VMware Cloud Director VM sizing policy data source. This can be
  used to read VM sizing policy.
---

# vcd\_vm\_sizing\_policy

Provides a VMware Cloud Director VM sizing policy data source. This can be
used to read VM sizing policy.

Supported in provider *v3.0+* and requires VCD 10.0+

## Example Usage

```hcl
data "vcd_vm_sizing_policy" "tf-policy-name" {
  name = "my-rule"
}
output "policyId" {
  value = data.vcd_vm_sizing_policy.tf-policy-name.id
}
```
## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `name` - (Required) The name VM sizing policy

All arguments defined in [`vcd_vm_sizing_policy`](/docs/providers/vcd/r/vm_sizing_policy.html#argument-reference) are supported.

