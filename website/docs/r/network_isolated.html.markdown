---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_network_isolated"
sidebar_current: "docs-vcd-resource-network-isolated"
description: |-
  Provides a VMware Cloud Director Org VDC isolated Network. This can be used to create, modify, and delete internal networks for vApps to connect.
---

# vcd\_network\_isolated

Provides a VMware Cloud Director Org VDC isolated Network. This can be used to create,
modify, and delete internal networks for vApps to connect. This network is not attached to external networks or routers.

Supported in provider *v2.0+*

~> **Note:** This resource supports only NSX-V backed Org VDC networks.
Please use newer [`vcd_network_isolated_v2`](/docs/providers/vcd/r/network_isolated_v2.html) resource
which is compatible with NSX-T.

## Example Usage

```hcl
resource "vcd_network_isolated" "net" {
  org = "my-org" # Optional
  vdc = "my-vdc" # Optional

  name    = "my-net"
  gateway = "192.168.2.1"
  dns1    = "192.168.2.1"

  dhcp_pool {
    start_address = "192.168.2.2"
    end_address   = "192.168.2.50"
  }

  static_ip_pool {
    start_address = "192.168.2.51"
    end_address   = "192.168.2.100"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional; *v2.0+*) The name of organization to use, optional if defined at provider level. Useful when
  connected as sysadmin working across different organisations
* `vdc` - (Optional; *v2.0+*) The name of VDC to use, optional if defined at provider level
* `name` - (Required) A unique name for the network
* `description` - (Optional *v2.6+*) An optional description of the network
* `netmask` - (Optional) The netmask for the new network. Defaults to `255.255.255.0`
* `gateway` (Required) The gateway for this network
* `dns1` - (Optional) First DNS server to use.
* `dns2` - (Optional) Second DNS server to use.
* `dns_suffix` - (Optional) A FQDN for the virtual machines on this network
* `shared` - (Optional) Defines if this network is shared between multiple VDCs
  in the Org.  Defaults to `false`.
* `dhcp_pool` - (Optional) A range of IPs to issue to virtual machines that don't
  have a static IP; see [IP Pools](#ip-pools) below for details.
* `static_ip_pool` - (Optional) A range of IPs permitted to be used as static IPs for
  virtual machines; see [IP Pools](#ip-pools) below for details.

<a id="ip-pools"></a>
## IP Pools

Static IP Pools and DHCP Pools support the following attributes:

* `start_address` - (Required) The first address in the IP Range
* `end_address` - (Required) The final address in the IP Range

DHCP Pools additionally support the following attributes:

* `default_lease_time` - (Optional) The default DHCP lease time to use. Defaults to `3600`.
* `max_lease_time` - (Optional) The maximum DHCP lease time to use. Defaults to `7200`.

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing isolated network can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of orgName.vdcName.networkName.
For example, using this structure, representing an isolated network that was **not** created using Terraform:

```hcl
resource "vcd_network_isolated" "tf-mynet" {
  name    = "my-net"
  org     = "my-org"
  vdc     = "my-vdc"
  gateway = "COMPUTE"
}
```

You can import such isolated network into terraform state using this command

```
terraform import vcd_network_isolated.tf-mynet my-org.my-vdc.my-net
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.
