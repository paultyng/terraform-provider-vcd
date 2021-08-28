---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxv_dhcp_relay"
sidebar_current: "docs-vcd-resource-nsxv-dhcp-relay"
description: |-
  Provides an NSX edge gateway DHCP relay configuration resource.
---

# vcd\_nsxv\_dhcp\_relay

Provides a VMware Cloud Director Edge Gateway DHCP relay configuration resource. The DHCP relay capability
provided by NSX in VMware Cloud Director environment allows to leverage existing DHCP infrastructure from
within VMware Cloud Director environment without any interruption to the IP address management in existing
DHCP infrastructure. DHCP messages are relayed from virtual machines to the designated DHCP servers
in your physical DHCP infrastructure, which allows IP addresses controlled by the NSX software to
continue to be in sync with IP addresses in the rest of your DHCP-controlled environments. 

~> **Note:** This resource is a "singleton". Because DHCP relay settings are just edge gateway
properties - only one resource per Edge Gateway is useful.

Supported in provider *v2.6+*

## Example Usage 1 (Minimal configuration)

```hcl
resource "vcd_nsxv_dhcp_relay" "relay_config" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  ip_addresses = ["1.1.1.1"]

  relay_agent {
    network_name = vcd_network_routed.test-routed[0].name
  }
}
```

## Example Usage 2 (Example of configuration with multiple relay agents)

```hcl
resource "vcd_nsxv_dhcp_relay" "relay_config" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  ip_addresses = ["1.1.1.1", "2.2.2.2"]
  domain_names = ["servergroups.domainname.com", "other.domain.com"]
  ip_sets      = [vcd_nsxv_ip_set.myset1.name, vcd_nsxv_ip_set.myset2.name]

  relay_agent {
    network_name = "my-routed-network-1"
  }

  relay_agent {
    network_name       = vcd_network_routed.db-network.name
    gateway_ip_address = "10.201.1.1"
  }
}

resource "vcd_nsxv_ip_set" "myset1" {
  org = "my-org"
  vdc = "my-org-vdc"

  name         = "ipset-one"
  ip_addresses = ["10.10.10.1/24"]
}

resource "vcd_nsxv_ip_set" "myset2" {
  org = "my-org"
  vdc = "my-org-vdc"

  name         = "ipset-two"
  ip_addresses = ["20.20.20.1/24"]
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway` - (Required) The name of the edge gateway on which DHCP relay is to be configured.
* `ip_addresses` - (Optional) A set of IP addresses.
* `domain_names` - (Optional) A set of domain names.
* `ip_sets` - (Optional) A set of IP set names.
* `relay_agent` - (Required) One or more blocks to define Org network and optional IP address of
  edge gateway interfaces from which DHCP messages are to be relayed to the external DHCP relay
  server(s). See [Relay Agent](#relay-agent) and example for usage details.

<a id="relay-agent"></a>
## Relay Agent

* `network_name` - (Required) An existing Org network name from which DHCP messages are to be relayed.
* `gateway_ip_address` - (Optional) IP address on edge gateway to be used for relaying messages.
  Primary address of edge gateway interface will be picked if not specified. 

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing DHCP relay configuration can be [imported][docs-import] into this resource
via supplying the full dot separated path for your edge gateway. An example is
below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_nsxv_dhcp_relay.imported my-org.my-org-vdc.my-edge-gw
```

The above would import the DHCP relay settings that are defined on edge
gateway `my-edge-gw` which is configured in organization named `my-org` and vDC named `my-org-vdc`.
