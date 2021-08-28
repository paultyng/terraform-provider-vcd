---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_independent_disk"
sidebar_current: "docs-vcd-data-source-independent-disk"
description: |-
  Provides a independent disk data source.
---

# vcd\_independent\_disk

Provides a VMware Cloud Director Independent disk data source. A independent disk data source can be used to reference an independent disk and use its 
data within other resources or data sources.

Supported in provider *v2.5+*

## Example Usage

```hcl
data "vcd_independent_disk" "existing-disk" {
  org  = "my-org"
  vdc  = "my-vdc"
  id   = "urn:vcloud:disk:1bbc273d-7701-4f06-97be-428b46b0805e"
  name = "my-disk"
}
output "disk-iops" {
  value = data.vcd_independent_disk.existing-disk.iops
}
output "type_is_attached" {
  value = data.vcd_independent_disk.existing-disk.is_attached
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `id` - (Optional) Disk id or name is required. If both provided - Id is used. Id can be found by using import function [Listing independent disk IDs](/docs/providers/vcd/r/independent_disk.html#listing-independent-disk-ids) 
* `name` - (Optional) Disk name.  **Warning** please use `id` as there is possibility to have more than one independent disk with same name. As result data source will fail.

## Attribute reference

All attributes defined in [independent disk](/docs/providers/vcd/r/independent_disk.html#attribute-reference) are supported.
