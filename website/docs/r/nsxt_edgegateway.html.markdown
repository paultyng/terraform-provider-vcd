---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_edgegateway"
sidebar_current: "docs-vcd-resource-nsxt-edge-gateway"
description: |-
  Provides a VMware Cloud Director NSX-T edge gateway. This can be used to create, update, and delete NSX-T edge gateways connected to external networks.
---

# vcd\_nsxt\_edgegateway

Provides a VMware Cloud Director NSX-T edge gateway. This can be used to create, update, and delete NSX-T edge gateways connected
to external networks.

~> **Note:** Only `System Administrator` can create an edge gateway.
You must use `System Adminstrator` account in `provider` configuration
and then provide `org` and `vdc` arguments for edge gateway to work.

-> **Note:** This resource uses new VMware Cloud Director
[OpenAPI](https://code.vmware.com/docs/11982/getting-started-with-vmware-cloud-director-openapi) and
requires at least VCD *10.1.1+* and NSX-T *3.0+*.

Supported in provider *v3.1+*.

## Example Usage (Simple case)

```hcl

data "vcd_external_network_v2" "nsxt-ext-net" {
  name = "nsxt-edge"
}

resource "vcd_nsxt_edgegateway" "nsxt-edge" {
  org         = "my-org"
  vdc         = "nsxt-vdc"
  name        = "nsxt-edge"
  description = "Description"

  external_network_id = data.vcd_external_network_v2.nsxt-ext-net.id

  subnet {
    gateway       = "10.150.191.253"
    prefix_length = "19"
    # primary_ip should fall into defined "allocated_ips" range as otherwise
    # next apply will report additional range of "allocated_ips" with the range
    # containing single "primary_ip" and will cause non-empty plan.
    primary_ip = "10.150.160.137"
    allocated_ips {
      start_address = "10.150.160.137"
      end_address   = "10.150.160.138"
    }
  }
}
```

## Example Usage (Using custom Edge Cluster and multiple subnets)

```hcl
data "vcd_nsxt_edge_cluster" "secondary" {
  name = "edge-cluster-two"
}


data "vcd_external_network_v2" "nsxt-ext-net" {
  name = "nsxt-edge"
}

resource "vcd_nsxt_edgegateway" "nsxt-edge" {
  org         = "my-org"
  vdc         = "nsxt-vdc"
  name        = "nsxt-edge"
  description = "Description"

  external_network_id       = data.vcd_external_network_v2.nsxt-ext-net.id
  dedicate_external_network = true

  # Custom edge cluster reference
  edge_cluster_id = data.vcd_nsxt_edge_cluster.secondary.id

  subnet {
    gateway       = "10.150.191.253"
    prefix_length = "19"
    # primary_ip should fall into defined "allocated_ips" range as otherwise
    # next apply will report additional range of "allocated_ips" with the range
    # containing single "primary_ip" and will cause non-empty plan.
    primary_ip = "10.150.160.137"
    allocated_ips {
      start_address = "10.150.160.137"
      end_address   = "10.150.160.137"
    }
  }

  subnet {
    gateway       = "77.77.77.1"
    prefix_length = "26"

    allocated_ips {
      start_address = "77.77.77.10"
      end_address   = "77.77.77.12"
    }
  }

  subnet {
    gateway       = "88.88.88.1"
    prefix_length = "24"

    allocated_ips {
      start_address = "88.88.88.91"
      end_address   = "88.88.88.92"
    }

    allocated_ips {
      start_address = "88.88.88.94"
      end_address   = "88.88.88.95"
    }

    allocated_ips {
      start_address = "88.88.88.97"
      end_address   = "88.88.88.98"
    }
  }
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the VDC belongs. Optional if defined at provider level.
* `vdc` - (Optional) The name of VDC that owns the edge gateway. Optional if defined at provider level.
* `name` - (Required) A unique name for the edge gateway.
* `description` - (Optional) A unique name for the edge gateway.
* `external_network_id` - (Required) An external network ID. **Note.** Data source [vcd_external_network_v2](/docs/providers/vcd/d/external_network_v2.html)
can be used to lookup ID by name.
* `subnet` - (Required) One or more [subnets](#edgegateway-subnet) defined for edge gateway.
* `edge_cluster_id` - (Optional) Specific Edge Cluster ID if required
* `dedicate_external_network` - (Optional) Dedicating the External Network will enable Route Advertisement for this Edge Gateway. Default `false`.

<a id="edgegateway-subnet"></a>
## Edge Gateway Subnet

* `gateway` (Required) - Gateway for a subnet in external network
* `prefix_length` (Required) - Prefix length of a subnet in external network (e.g. 24 for netmask of 255.255.255.0)
* `primary_ip` (Optional) - Primary IP address for edge gateway. **Note:** `primary_ip` must fall into `allocated_ips`
block range as otherwise `plan` will not be clean with a new range defined for that particular block. There __can only
be one__ `primary_ip` defined for edge gateway.
* `allocated_ips` (Required) - One or more blocks of [ip ranges](#edgegateway-subnet-ip-allocation) in the subnet to be
allocated

<a id="edgegateway-subnet-ip-allocation"></a>
## Edge Gateway Subnet IP Allocation

* `start_address` (Required) - Start IP address of a range
* `end_address` (Required) - End IP address of a range


## Attribute Reference

The following attributes are exported on this resource:

* `primary_ip` - Primary IP address exposed for an easy access without nesting.


## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing edge gateway can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of org-name.vdc-name.nsxt-edge-name
For example, using this structure, representing an edge gateway that was **not** created using Terraform:

```hcl
resource "vcd_nsxt_edgegateway" "nsxt-edge" {
  org         = "my-org"
  vdc         = "nsxt-vdc"
  name        = "nsxt-edge"
  description = "Description"

  external_network_id = data.vcd_external_network_v2.nsxt-ext-net.id

  subnet {
    gateway       = "10.10.10.1"
    prefix_length = "24"
    primary_ip    = "10.10.10.10"
    allocated_ips {
      start_address = "10.10.10.10"
      end_address   = "10.10.10.30"
    }
  }
}
```

You can import such resource into terraform state using the command below:

```
terraform import vcd_nsxt_edgegateway.nsxt-edge my-org.nsxt-vdc.nsxt-edge
```

* **Note 1**: the separator can be changed using `Provider.import_separator` or variable `VCD_IMPORT_SEPARATOR`
* **Note 2**: it is possible to list all available NSX-T edge gateways using data source [vcd_resource_list](/docs/providers/vcd/d/resource_list.html#vcd_nsxt_edgegateway)

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.
