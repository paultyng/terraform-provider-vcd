---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_edgegateway"
sidebar_current: "docs-vcd-resource-edgegateway"
description: |-
  Provides a VMware Cloud Director edge gateway. This can be used to create and delete edge gateways connected to one or more external networks.
---

# vcd\_edgegateway

Provides a VMware Cloud Director edge gateway directly connected to one or more external networks. This can be used to create
and delete edge gateways for Org VDC networks to connect.

Supported in provider *v2.4+*

~> **Note:** Only `System Administrator` can create an edge gateway.
You must use `System Adminstrator` account in `provider` configuration
and then provide `org` and `vdc` arguments for edge gateway to work.

## Example Usage

```hcl
resource "vcd_edgegateway" "egw" {
  org = "my-org"
  vdc = "my-vdc"

  name          = "my-egw"
  description   = "new edge gateway"
  configuration = "compact"

  external_network {
    name = "my-ext-net1"

    subnet {
      ip_address            = "192.168.30.51"
      gateway               = "192.168.30.49"
      netmask               = "255.255.255.240"
      use_for_default_route = true
    }
  }
}

resource "vcd_network_routed" "rnet1" {
  name         = "rnet1"
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = vcd_edgegateway.egw.name
  gateway      = "192.168.2.1"

  static_ip_pool {
    start_address = "192.168.2.2"
    end_address   = "192.168.2.100"
  }
}
```


## Example Usage (multiple External Networks, Subnets and IP pool sub-allocation)

```hcl
resource "vcd_edgegateway" "egw" {
  org = "my-org"
  vdc = "my-vdc"

  name          = "edge-with-complex-networks"
  description   = "new edge gateway"
  configuration = "compact"

  external_network {
    name = "my-main-external-network"

    subnet {
      ip_address = "192.168.30.51"
      gateway    = "192.168.30.49"
      netmask    = "255.255.255.240"

      suballocate_pool {
        start_address = "192.168.30.53"
        end_address   = "192.168.30.55"
      }

      suballocate_pool {
        start_address = "192.168.30.58"
        end_address   = "192.168.30.60"
      }
    }

    subnet {
      # ip_address is skipped here on purpose to get dynamic IP assigned. Because this
      # subnet is used for default route, this IP address can then be accessed using
      # `default_external_network_ip` attribute.
      use_for_default_route = true
      gateway               = "192.168.40.149"
      netmask               = "255.255.255.0"
    }
  }

  external_network {
    name = "my-other-external-network"

    subnet {
      # IP address will be auto-assigned. It can then be found in the list of `external_network_ips`
      # attribute
      gateway    = "1.1.1.1"
      netmask    = "255.255.255.248"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the VDC belongs. Optional if defined at provider level.
* `vdc` - (Optional) The name of VDC that owns the edge gateway. Optional if defined at provider level. 
* `name` - (Required) A unique name for the edge gateway.
* `external_network` - (Required, *v2.6+*) One or more blocks defining external networks, their
  subnets, IP addresses and  IP pool suballocation attached to edge gateway interfaces. Details are
  in [external network](#external-network) block below.
* `configuration` - (Required) Configuration of the vShield edge VM for this gateway. One of: `compact`, `full` ("Large"), `x-large`, `full4` ("Quad Large").
* `ha_enabled` - (Optional) Enable high availability on this edge gateway. Default is `false`.
* `distributed_routing` - (Optional) If advanced networking enabled, also enable distributed
  routing. Default is `false`.
* `fips_mode_enabled` - (Optional) When FIPS mode is enabled, any secure communication to or from
  the NSX Edge uses cryptographic algorithms or protocols that are allowed by United States Federal
  Information Processing Standards (FIPS). FIPS mode turns on the cipher suites that comply with
  FIPS. Default is `false`. **Note:** to use FIPS mode it must be enabled in vCD system settings.
* `use_default_route_for_dns_relay` - (Optional) When default route is set, it will be used for
  gateways' default routing and DNS forwarding. Default is `false`.
* `lb_enabled` - (Optional) Enable load balancing. Default is `false`.
* `lb_acceleration_enabled` - (Optional) Enable to configure the load balancer to use the faster L4
engine rather than L7 engine. The L4 TCP VIP is processed before the edge gateway firewall so no 
`allow` firewall rule is required. Default is `false`. **Note:** L7 VIPs for HTTP and HTTPS are
processed after the firewall, so when Acceleration Enabled is not selected, an edge gateway firewall
rule must exist to allow access to the L7 VIP for those protocols. When Acceleration Enabled is
selected and the server pool is in non-transparent mode, an SNAT rule is added, so you must ensure
that the firewall is enabled on the edge gateway.
* `lb_logging_enabled` - (Optional) Enables the edge gateway load balancer to collect traffic logs.
Default is `false`.
* `lb_loglevel` - (Optional) Choose the severity of events to be logged. One of `emergency`,
`alert`, `critical`, `error`, `warning`, `notice`, `info`, `debug`
* `fw_enabled` (Optional) Enable firewall. Default `true`. **Note:** Disabling Firewall will also
disable NAT and other NAT dependent features like Load Balancer.
* `fw_default_rule_logging_enabled` (Optional) Enable default firewall rule (last in the processing 
order) logging. Default `false`.
* `fw_default_rule_action` (Optional) Default firewall rule (last in the processing order) action.
One of `accept` or `deny`. Default `deny`.

<a id="external-network"></a>
## External Network

* `name` (Required) - Name of existing external network
* `enable_rate_limit` (Optional) - `True` if rate limiting should be applied on this interface.
  Default is `false`.
* `incoming_rate_limit` (Optional) - Incoming rate limit in Mbps.
* `outgoing_rate_limit` (Optional) - Outgoing rate limit in Mbps.
* `subnet` (Required) - One or more blocks of [External Network Subnet](#external-network-subnet).

~> **Note:** Rate limiting works only with external networks backed by distributed portgroups.


<a id="external-network-subnet"></a>
## External Network Subnet 

* `gateway` (Required) - Gateway for a subnet in external network
* `netmask` (Required) - Netmask of a subnet in external network
* `ip_address` (Optional) - IP address to assign to edge gateway interface (will be auto-assigned if
  unspecified)
* `use_for_default_route` (Optional) - Should this network be used as default gateway on edge
  gateway. Default is `false`. 
* `suballocate_pool` (Optional) - One or more blocks of [ip
  ranges](#external-network-subnet-suballocate) in the subnet to be sub-allocated 

<a id="external-network-subnet"></a>
## External Network Subnet Sub-Allocation

* `start_address` (Required) - Start IP address of a range
* `end_address` (Required) - End IP address of a range


## Attribute Reference

The following attributes are exported on this resource:

* `default_external_network_ip` (*v2.6+*) - IP address of edge gateway used for default network
* `external_network_ips` (*v2.6+*) - A list of IP addresses assigned to edge gateway interfaces
  connected to external networks.


## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing edge gateway can be [imported][docs-import] into this resource via supplying its path. 
The path for this resource is made of org-name.vdc-name.edge-name
For example, using this structure, representing an edge gateway that was **not** created using Terraform:

```hcl
resource "vcd_edgegateway" "tf-edgegateway" {
  name          = "my-edge-gw"
  org           = "my-org"
  vdc           = "my-vdc"
  configuration = "COMPUTE"

  external_network {
    name = "my-ext-net1"

    subnet {
      ip_address            = "192.168.30.51"
      gateway               = "192.168.30.49"
      netmask               = "255.255.255.240"
      use_for_default_route = true
    }
  }
}
```

You can import such resource into terraform state using one of the commands below

```
terraform import vcd_edgegateway.tf-egw my-org.my-vdc.my-edge-gw

terraform import vcd_edgegateway.tf-egw my-org.my-vdc.63ed92de-4001-450c-879f-deadbeef0123
```

* **Note 1**: the separator can be changed using `Provider.import_separator` or variable `VCD_IMPORT_SEPARATOR`
* **Note 2**: the identifier of the resource could be either the edge gateway name or the ID

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for 
further operations.
