---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_storage_profile"
sidebar_current: "docs-vcd-data-source-storage-profile"
description: |-
  Provides a data source for VDC storage profile.
---

# vcd\_storage\_profile

Provides a data source for VDC storage profile.

Supported in provider *v3.1+*


## Example Usage

```hcl
data "vcd_storage_profile" "sp" {
  org  = "my-org"
  vdc  = "my-vdc"
  name = "ssd-storage-profile"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `name` - (Required) Storage profile name.

## Attribute reference

This data source exports only `id` field.
