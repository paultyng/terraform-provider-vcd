---
layout: "vcd"
page_title: "VMware Cloud Foundation Tenant Manager: vcd_tm_storage_class"
sidebar_current: "docs-vcd-data-source-tm-storage-class"
description: |-
  Provides a VMware Cloud Foundation Tenant Manager data source to read Storage Classes.
---

# vcd\_tm\_storage\_class

Provides a VMware Cloud Foundation Tenant Manager data source to read Region Storage Classes.

This data source is exclusive to **VMware Cloud Foundation Tenant Manager**. Supported in provider *v4.0+*

## Example Usage

```hcl
data "vcd_tm_region" "region" {
  name = "my-region"
}

data "vcd_tm_region_storage_class" "sc" {
  region_id = data.vcd_tm_region.region.id
  name      = "vSAN Default Storage Class"
}

resource "vcd_tm_content_library" "cl" {
  name        = "My Library"
  description = "A simple library"
  storage_class_ids = [
    data.vcd_tm_storage_class.sc.id
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Storage Class to read
* `region_id` - (Required) The ID of the Region where the Storage Class belongs

## Attribute reference

* `storage_capacity_mib` - The total storage capacity of the Storage Class in mebibytes
* `storage_consumed_mib` - For tenants, this represents the total storage given to all namespaces consuming from this
  Storage Class in mebibytes. For providers, this represents the total storage given to tenants from this Storage Class
  in mebibytes
* `zone_ids` - A set with all the IDs of the zones available to the Storage Class