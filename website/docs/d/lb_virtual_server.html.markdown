---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_lb_virtual_server"
sidebar_current: "docs-vcd-data-source-lb-virtual-server"
description: |-
  Provides an NSX edge gateway load balancer virtual server data source.
---

# vcd\_lb\_virtual\_server

Provides a VMware Cloud Director edge gateway load balancer virtual server data source. Adds an edge gateway
internal or uplink interface as a virtual server. A virtual server has a public IP address and services all incoming client requests. 

~> **Note:** To make load balancing work one must ensure that load balancing is enabled on edge gateway (edge gateway must be advanced).
This depends on NSX version to work properly. Please refer to [VMware Product Interoperability Matrices](https://www.vmware.com/resources/compatibility/sim/interop_matrix.php#interop&29=&93=) 
to check supported VMware Cloud Director and NSX for vSphere configurations.

~> **Note:** The VMware Cloud Director API for NSX supports a subset of the operations and objects defined in the NSX vSphere 
API Guide. The API supports NSX 6.2, 6.3, and 6.4.

Supported in provider *v2.4+*

## Example Usage

```hcl
data "vcd_lb_virtual_server" "my-vs" {
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
* `edge_gateway` - (Required) The name of the edge gateway on which the virtual server is defined
* `name` - (Required) Name for identifying the exact virtual server

## Attribute Reference

All the attributes defined in `vcd_lb_virtual_server` resource are available.
