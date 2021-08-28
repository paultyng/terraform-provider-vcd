---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_tier0_router"
sidebar_current: "docs-vcd-data-source-nsxt-tier0-router"
description: |-
  Provides a data source for available NSX-T Tier-0 routers.
---

# vcd\_nsxt\_tier0\_router

Provides a data source for available NSX-T Tier-0 routers.

Supported in provider *v3.0+*

~> **Note:** This resource uses new VMware Cloud Director
[OpenAPI](https://code.vmware.com/docs/11982/getting-started-with-vmware-cloud-director-openapi) and
requires at least VCD *10.1.1+* and NSX-T *3.0+*.

## Example Usage 

```hcl
data "vcd_nsxt_manager" "main" {
  name = "nsxt-manager-one"
}

data "vcd_nsxt_tier0_router" "router" {
  name            = "nsxt-tier0-router"
  nsxt_manager_id = data.vcd_nsxt_manager.main.id
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) NSX-T Tier-0 router name. **Note**. Tier-0 router name must be unique inside NSX-T manager because
API does not allow to filter by other fields.
* `nsxt_manager_id` - (Required) NSX-T manager should be referenced.

## Attribute reference

* `is_assigned` - Boolean value reflecting if Tier-0 router is already consumed by external network.
