---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_lb_server_pool"
sidebar_current: "docs-vcd-data-source-lb-server-pool"
description: |-
  Provides an NSX edge gateway load balancer server pool data source.
---

# vcd\_lb\_server\_pool

Provides a VMware Cloud Director Edge Gateway Load Balancer Server Pool data source. A Server Pool defines
a group of backend servers (defined as pool members), manages load balancer distribution methods, and has a service 
monitor attached to it for health check parameters.

~> **Note:** See additional support notes in [server pool resource page](/docs/providers/vcd/r/lb_server_pool.html).

Supported in provider *v2.4+*

## Example Usage

```hcl
data "vcd_lb_server_pool" "sp-ds" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  name = "not-managed"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `edge_gateway` - (Required) The name of the edge gateway on which the server pool is defined
* `name` - (Required) Server Pool name for identifying the exact server pool

## Attribute Reference

All the attributes defined in `vcd_lb_server_pool` resource are available.
