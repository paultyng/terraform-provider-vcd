---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_lb_server_pool"
sidebar_current: "docs-vcd-resource-lb-server-pool"
description: |-
  Provides an NSX edge gateway load balancer server pool resource.
---

# vcd\_lb\_server\_pool

Provides a VMware Cloud Director Edge Gateway Load Balancer Server Pool resource. A Server Pool can have a group of backend
servers set (defined as pool members), manages load balancer distribution methods, and may have a service monitor
attached to it for health check parameters.

~> **Note:** To make load balancing work one must ensure that load balancing is enabled on edge gateway. This depends 
on NSX version to work properly. Please refer to [VMware Product Interoperability Matrices](https://www.vmware.com/resources/compatibility/sim/interop_matrix.php#interop&29=&93=) 
to check supported VMware Cloud Director and NSX for vSphere configurations.

~> **Note:** The VMware Cloud Director API for NSX supports a subset of the operations and objects defined in the NSX vSphere 
API Guide. The API supports NSX 6.2, 6.3, and 6.4.

Supported in provider *v2.4+*

## Example Usage 1 (Simple Server Pool without Service Monitor)

```hcl
resource "vcd_lb_server_pool" "web-servers" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  name      = "web-servers"
  algorithm = "round-robin"

  member {
    condition       = "enabled"
    name            = "member1"
    ip_address      = "1.1.1.1"
    port            = 8443
    monitor_port    = 9000
    weight          = 1
    min_connections = 0
    max_connections = 100
  }
}
```

## Example Usage 2 (Server Pool with multiple members, algorithm parameters, and existing Service Monitor as data source)

```hcl
data "vcd_lb_service_monitor" "web-monitor" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  name = "existing-web-monitor-name"
}

resource "vcd_lb_server_pool" "web-servers" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  name                 = "web-servers"
  description          = "description"
  algorithm            = "httpheader"
  algorithm_parameters = "headerName=host"
  enable_transparency  = "true"

  monitor_id = data.vcd_lb_service_monitor.web-monitor.id

  member {
    condition       = "enabled"
    name            = "member1"
    ip_address      = "1.1.1.1"
    port            = 8443
    monitor_port    = 9000
    weight          = 1
    min_connections = 0
    max_connections = 100
  }

  member {
    condition       = "drain"
    name            = "member2"
    ip_address      = "2.2.2.2"
    port            = 7000
    monitor_port    = 4000
    weight          = 2
    min_connections = 6
    max_connections = 8
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `edge_gateway` - (Required) The name of the edge gateway on which the server pool is to be created
* `name` - (Required) Server Pool name
* `description` - (Optional) Server Pool description
* `algorithm` - (Required) Server Pool load balancing method. Can be one of `ip-hash`, `round-robin`, `uri`, `leastconn`, `url`, or `httpheader`
* `algorithm_parameters` - (Optional) Valid only when `algorithm` is `httpheader` or `url`. The `httpheader` algorithm
parameter has one option `headerName=<name>` while the `url` algorithm parameter has option `urlParam=<url>`. 
* `enable_transparency` - (Optional) When transparency is `false` (default) backend servers see the IP address of the
traffic source as the internal IP address of the load balancer. When it is `true` the source IP address is the actual IP
address of the client and the edge gateway must be set as the default gateway to ensure that return packets go through
the edge gateway. 
* `monitor_id` - (Optional) `vcd_lb_service_monitor` resource `id` to attach to server pool for health check parameters
* `member` - (Optional) A block to define server pool members. Multiple can be used. See [Member](#member) and 
example for usage details.


<a id="member"></a>
## Member

* `condition` - (Required) State of member in a pool. One of `enabled`, `disabled`, or `drain`. When member condition 
is set to `drain` it stops taking new connections and calls, while it allows its sessions on existing connections to
continue until they naturally end. This allows to gracefully remove member node from load balancing rotation.
* `name` - (Required) Member name
* `ip_address` - (Required) Member IP address
* `port` - (Required) The port at which the member is to receive traffic from the load balancer.
* `monitor_port` - (Required) Monitor Port at which the member is to receive health monitor requests. **Note:** can
be the same as `port`
* `weight` - (Required) The proportion of traffic this member is to handle. Must be an integer in the range 1-256.
* `max_connections` - (Optional) The maximum number of concurrent connections the member can handle. **Note:** when the
number of incoming requests exceeds the maximum, requests are queued and the load balancer waits for a connection to be
released. 
* `min_connections` - (Optional) The minimum number of concurrent connections a member must always accept.

## Attribute Reference

The following attributes are exported on this resource:

* `id` - The NSX ID of the load balancer server pool

Additionally each of members defined in blocks expose their own `id` fields as well

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.](https://www.terraform.io/docs/import/)

An existing load balancer server pool can be [imported][docs-import] into this resource
via supplying the full dot separated path for load balancer service monitor. An example is below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_lb_server_pool.imported my-org.my-org-vdc.my-edge-gw.my-lb-server-pool
```

The above would import the server pool named `my-lb-server-pool` that is defined on edge gateway
`my-edge-gw` which is configured in organization named `my-org` and vDC named `my-org-vdc`.
