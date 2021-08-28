---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_external_network"
sidebar_current: "docs-vcd-resource-external-network"
description: |-
  Provides a VMware Cloud Director external network resource.  This can be used to create and delete external networks.
---

# vcd\_external\_network

Provides a VMware Cloud Director external network resource.  This can be used to create and delete external networks.
Requires system administrator privileges.

Supported in provider *v2.2+*

~> **Note:** For NSX-T suported external network please use [vcd_external_network_v2](/docs/providers/vcd/r/external_network_v2.html)

## Example Usage

```hcl
provider "vcd" {
  user     = var.admin_user
  password = var.admin_password
  org      = "System"
  url      = "https://Vcd/api"
}

resource "vcd_external_network" "net" {
  name        = "my-ext-net"
  description = "Reference for vCD external network"

  ip_scope {
    gateway    = "192.168.30.49"
    netmask    = "255.255.255.240"
    dns1       = "192.168.0.164"
    dns2       = "192.168.0.196"
    dns_suffix = "mybiz.biz"

    static_ip_pool {
      start_address = "192.168.30.51"
      end_address   = "192.168.30.62"
    }
  }

  # It's possible to define more than one IP scope
  ip_scope {
    gateway    = "192.168.31.49"
    netmask    = "255.255.255.240"
    dns1       = "192.168.1.164"
    dns2       = "192.168.1.196"
    dns_suffix = "my.biz"

    static_ip_pool {
      start_address = "192.168.31.51"
      end_address   = "192.168.31.55"
    }

    static_ip_pool {
      start_address = "192.168.31.57"
      end_address   = "192.168.31.59"
    }
  }

  vsphere_network {
    name    = "myNetwork"
    type    = "DV_PORTGROUP"
    vcenter = "vcenter-name"
  }

  # It's possible to define more than one vSphere network
  vsphere_network {
    name    = "myNetwork2"
    type    = "DV_PORTGROUP"
    vcenter = "vcenter-name2"
  }

  retain_net_info_across_deployments = "false"
}

resource "vcd_network_direct" "net" {
  org              = "my-org"
  vdc              = "my-vdc"
  name             = "my-net"
  external_network = vcd_external_network.net.name
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the network
* `description` - (Optional) Network friendly description
* `ip_scope` - (Required) A list of IP scopes for the network.  See [IP Scope](#ipscope) below for details.
* `vsphere_network` - (Required) A list of DV_PORTGROUP or NETWORK objects names that back this network. Each referenced DV_PORTGROUP or NETWORK must exist on a vCenter server registered with the system.  See [vSphere Network](#vspherenetwork) below for details.
* `retain_net_info_across_deployments` - (Optional) Specifies whether the network resources such as IP/MAC of router will be retained across deployments. Default is false.

<a id="ipscope"></a>
## IP Scope

* `gateway` - (Required) Gateway of the network
* `netmask` - (Required) Network mask
* `dns1` - (Optional) Primary DNS server
* `dns2` - (Optional) Secondary DNS server
* `dns_suffix` (Optional) A FQDN for the virtual machines on this network.
* `static_ip_pool` - (Required) IP ranges used for static pool allocation in the network.  See [IP Pool](#ip-pool) below for details.

<a id="ip-pool"></a>
## IP Pool

* `start_address` - (Required) Start address of the IP range
* `end_address` - (Required) End address of the IP range

<a id="vspherenetwork"></a>
## vSphere Network

* `name` - (Required) Port group name
* `type` - (Required) The vSphere type of the object. One of: DV_PORTGROUP (distributed virtual port group), NETWORK (standard switch port group)
* `vcenter` - (Required) The vCenter server name

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing external network can be [imported][docs-import] into this resource via supplying the path for an external network. Since the external network is
at the top of the vCD hierarchy, the path corresponds to the external network name.
For example, using this structure, representing an existing external network that was **not** created using Terraform:

```hcl
resource "vcd_external_network" "tf-external-network" {
  name = "my-ext-net"
}
```

You can import such external network into terraform state using this command

```
terraform import vcd_external_network.tf-external-network my-ext-net
```

[docs-import]:https://www.terraform.io/docs/import/

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

While the above structure is the minimum needed to get an import, it is not sufficient to run `terraform plan`,
as it lacks several mandatory fields. To use the imported resource, you will need to add the missing properties
using the data in `terraform.tfstate` as a reference. If the resource does not need modifications, consider using
an [external network data source](/docs/providers/vcd/d/external_network.html) instead. 
