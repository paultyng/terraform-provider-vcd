---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxv_ip_set"
sidebar_current: "docs-vcd-resource-ipset"
description: |-
  Provides an IP set resource.
---

# vcd\_nsxv\_ip\_set

Provides a VMware Cloud Director IP set resource. An IP set is a group of IP addresses that you can add as
  the source or destination in a firewall rule or in DHCP relay configuration.


Supported in provider *v2.6+*

## Example Usage 1

```hcl
resource "vcd_nsxv_ip_set" "test-ipset" {
  org = "my-org"
  vdc = "my-org-vdc"

  name                   = "ipset-one"
  is_inheritance_allowed = false
  description            = "test-ip-set-changed-description"
  ip_addresses           = ["1.1.1.1/24", "10.10.10.100-10.10.10.110"]
}
```

## Example Usage 2 (minimal example)

```hcl
resource "vcd_nsxv_ip_set" "test-ipset" {
  name         = "ipset-two"
  ip_addresses = ["192.168.1.1"]
}
```

## Example Usage 3 (use IP set in firewall rules)

```hcl
resource "vcd_nsxv_ip_set" "test-ipset" {
  org = "my-org"
  vdc = "my-org-vdc"

  name                   = "ipset-one"
  is_inheritance_allowed = true
  description            = "test-ip-set-changed-description"
  ip_addresses           = ["1.1.1.1/24", "10.10.10.100-10.10.10.110"]
}

resource "vcd_nsxv_ip_set" "test-ipset2" {
  name         = "ipset-two"
  ip_addresses = ["192.168.1.1"]
}

resource "vcd_nsxv_firewall_rule" "ipsets" {
  org          = "my-org"
  vdc          = "my-org-vdc"
  edge_gateway = "my-edge-gw"

  name   = "rule-with-ipsets"
  action = "accept"

  source {
    ip_sets = [vcd_nsxv_ip_set.test-ipset.name]
  }

  destination {
    ip_sets = [vcd_nsxv_ip_set.test-ipset2.name]
  }

  service {
    protocol = "any"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `name` - (Required) Unique IP set name.
* `description` - (Optional) An optional description for IP set.
* `ip_addresses` - (Required) A set of IP addresses, CIDRs and ranges as strings.
* `is_inheritance_allowed` (Optional) Toggle to enable inheritance to allow visibility at underlying scopes. Default `true`

## Attribute Reference

The following attributes are exported on this resource:

* `id` - ID of IP set

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing IP set can be [imported][docs-import] into this resource via supplying the full dot
separated path IP set. An example is below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_nsxv_ip_set.imported org-name.vdc-name.ipset-name
```

The above would import the IP set named `ipset-name` that is defined in org named `org-name` and vDC
named `vdc-name`.
