---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_org_network"
sidebar_current: "docs-vcd-datasource-vapp-org-network"
description: |-
  Provides a data source for VMware Cloud Director Org network attached to vApp. This can be used to access vApp Org network.
---

# vcd\_vapp\_org\_network

Provides a data source for VMware Cloud Director Org network attached to vApp. This can be used to access vApp Org VDC network.

Supported in provider *v2.7+*

## Example Usage

```hcl

data "vcd_vapp" "web" {
  name = "web"
}

data "vcd_vapp_org_network" "network1" {
  vapp_name        = data.vcd_vapp.web.name
  org_network_name = "my-vapp-org-network"
}

output "id" {
  value = data.vcd_vapp_network.network1.id
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `vapp_name` - (Required) The vApp name.
* `org_network_name` - (Required) A name for the vApp Org network, unique within the vApp.

## Attribute reference

All attributes defined in [`vcd_vapp_org_network`](/docs/providers/vcd/r/vapp_org_network.html#attribute-reference) are supported.

