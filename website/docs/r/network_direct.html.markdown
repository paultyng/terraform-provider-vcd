---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_network_direct"
sidebar_current: "docs-vcd-resource-network-direct"
description: |-
  Provides a VMware Cloud Director Org VDC Network attached to an external one. This can be used to create, modify, and delete internal networks for vApps to connect.
---

# vcd\_network\_direct

Provides a VMware Cloud Director Org VDC Network directly connected to an external network. This can be used to create,
modify, and delete internal networks for vApps to connect.

Supported in provider *v2.0+*

~> **Note:** Only `System Administrator` can create an organization virtual datacenter network that connects
directly to an external network. You must use `System Adminstrator` account in `provider` configuration
and then provide `org` and `vdc` arguments for direct networks to work.

## Example Usage

```hcl
resource "vcd_network_direct" "net" {
  org = "my-org" # Optional
  vdc = "my-vdc" # Optional

  name             = "my-net"
  external_network = "my-ext-net"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional; *v2.0+*) The name of organization to use, optional if defined at provider level. Useful when
  connected as sysadmin working across different organisations
* `vdc` - (Optional; *v2.0+*) The name of VDC to use, optional if defined at provider level
* `name` - (Required) A unique name for the network
* `description` - (Optional *v2.6+*) An optional description of the network
* `external_network` - (Required) The name of the external network.
* `shared` - (Optional) Defines if this network is shared between multiple VDCs
  in the Org.  Defaults to `false`.

## Attribute reference

Supported in provider *v2.5+*

* `external_network_gateway` - (Computed) returns the gateway from the external network
* `external_network_netmask` - (Computed) returns the netmask from the external network
* `external_network_dns1` - (Computed) returns the first DNS from the external network
* `external_network_dns2` - (Computed) returns the second DNS from the external network
* `external_network_dns_suffix` - (Computed) returns the DNS suffix from the external network

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing direct network can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of orgName.vdcName.networkName.
For example, using this structure, representing a direct network that was **not** created using Terraform:

```hcl
resource "vcd_network_direct" "tf-mynet" {
  name             = "my-net"
  org              = "my-org"
  vdc              = "my-vdc"
  external_network = "COMPUTE"
}
```

You can import such isolated network into terraform state using this command

```
terraform import vcd_network_direct.tf-mynet my-org.my-vdc.my-net
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.
