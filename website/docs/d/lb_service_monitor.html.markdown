---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_lb_service_monitor"
sidebar_current: "docs-vcd-data-source-lb-service-monitor"
description: |-
  Provides an NSX edge gateway load balancer service monitor data source.
---

# vcd\_lb\_service\_monitor

Provides a VMware Cloud Director Edge Gateway Load Balancer Service Monitor data source. A service monitor 
defines health check parameters for a particular type of network traffic. It can be associated with
a pool. Pool members are monitored according to the service monitor parameters. See example usage of
this data source in [server pool resource page](/docs/providers/vcd/r/lb_server_pool.html).

~> **Note:** See additional support notes in [service monitor resource page](/docs/providers/vcd/r/lb_service_monitor.html).

Supported in provider *v2.4+*

## Example Usage

```hcl
data "vcd_lb_service_monitor" "my-monitor" {
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
* `edge_gateway` - (Required) The name of the edge gateway on which the service monitor is defined
* `name` - (Required) Service Monitor name for identifying the exact service monitor

## Attribute Reference

All the attributes defined in `vcd_lb_service_monitor` resource are available.
