---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_network"
sidebar_current: "docs-vcd-resource-vapp-network"
description: |-
 Allows to provision a vApp network and optionally connect it to an existing Org VDC network.
---

# vcd\_vapp\_network

 Allows to provision a vApp network and optionally connect it to an existing Org VDC network.

Supported in provider *v2.1+*

## Example Usage

```hcl
resource "vcd_vapp_network" "vappNet" {
  org = "my-org" # Optional
  vdc = "my-vdc" # Optional

  name               = "my-net"
  vapp_name          = "my-vapp"
  gateway            = "192.168.2.1"
  netmask            = "255.255.255.0"
  dns1               = "192.168.2.1"
  dns2               = "192.168.2.2"
  dns_suffix         = "mybiz.biz"
  guest_vlan_allowed = true

  static_ip_pool {
    start_address = "192.168.2.51"
    end_address   = "192.168.2.100"
  }

  dhcp_pool {
    start_address = "192.168.2.2"
    end_address   = "192.168.2.50"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional; *v2.0+*) The name of organization to use, optional if defined at provider level. Useful when 
  connected as sysadmin working across different organisations.
* `vdc` - (Optional; *v2.0+*) The name of VDC to use, optional if defined at provider level.
* `name` - (Required) A unique name for the network.
* `description` - (Optional; *v2.7+*, *vCD 9.5+*) Description of vApp network
* `vapp_name` - (Required) The vApp this network belongs to.
* `netmask` - (Optional) The netmask for the new network. Default is `255.255.255.0`.
* `gateway` (Required) The gateway for this network.
* `dns1` - (Optional) First DNS server to use.
* `dns2` - (Optional) Second DNS server to use.
* `dns_suffix` - (Optional) A FQDN for the virtual machines on this network.
* `guest_vlan_allowed` (Optional) True if Network allows guest VLAN tagging.
* `static_ip_pool` - (Optional) A range of IPs permitted to be used as static IPs for virtual machines; see [IP Pools](#ip-pools) below for details.
* `dhcp_pool` - (Optional) A range of IPs to issue to virtual machines that don't have a static IP; see [IP Pools](#ip-pools) below for details.
* `org_network_name` - (Optional; *v2.7+*) An Org network name to which vApp network is connected. If not configured, then an isolated network is created.
* `retain_ip_mac_enabled` - (Optional; *v2.7+*) Specifies whether the network resources such as IP/MAC of router will be retained across deployments. Default is false.

<a id="ip-pools"></a>
## IP Pools

Static IP Pools and DHCP Pools support the following attributes:

* `start_address` - (Required) The first address in the IP Range.
* `end_address` - (Required) The final address in the IP Range.

DHCP Pools additionally support the following attributes:

* `default_lease_time` - (Optional) The default DHCP lease time to use. Defaults to `3600`.
* `max_lease_time` - (Optional) The maximum DHCP lease time to use. Defaults to `7200`.
* `enabled` - (Optional) Allows to enable or disable service. Default is true.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing vApp Network can be [imported][docs-import] into this resource
via supplying the full dot separated path for vApp Network. An example is below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_vapp_network.imported org-name.vdc-name.vapp-name.network-name
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

The above command would import the vApp Network named `network-name` that is defined on vApp `vapp-name` 
which is configured in organization named `my-org` and VDC named `my-org-vdc`.
