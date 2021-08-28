---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_network"
sidebar_current: "docs-vcd-datasource-vapp-network"
description: |-
  Provides a VMware Cloud Director vApp network data source. This can be used to access a vApp network.
---

# vcd\_vapp\_network

Provides a VMware Cloud Director vApp network data source. This can be used to access a vApp network.

Supported in provider *v2.7+*

## Example Usage

```hcl

data "vcd_vapp" "web" {
  name = "web"
}

data "vcd_vapp_network" "network1" {
  vapp_name = data.vcd_vapp.web.name
  name      = "isolated-network"
}

output "gateway" {
  value = data.vcd_vapp_network.network1.gateway
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `vapp_name` - (Required) The vApp name.
* `name` - (Required) A name for the vApp network, unique within the vApp 

## Attribute reference

All attributes defined in [`vcd_vapp_network`](/docs/providers/vcd/r/vapp_network.html#attribute-reference) are supported.

