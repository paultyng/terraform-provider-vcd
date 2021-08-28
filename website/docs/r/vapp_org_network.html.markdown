---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_org_network"
sidebar_current: "docs-vcd-resource-vapp-org-network"
description: |-
  Provides capability to attach an existing Org VDC Network to a vApp and toggle network features.
---

# vcd\_vapp\_org\_network

 Provides capability to attach an existing Org VDC Network to a vApp and toggle network features.

Supported in provider *v2.7+*

## Example Usage

```hcl
resource "vcd_vapp_org_network" "vappOrgNet" {
  org = "my-org" # Optional
  vdc = "my-vdc" # Optional

  vapp_name = "my-vapp"

  # Comment below line to create an isolated vApp network
  org_network_name = "my-org-network"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when 
  connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `vapp_name` - (Required) The vApp this network belongs to.
* `org_network_name` - (Optional; *v2.7+*) An Org network name to which vApp network is connected. If not configured, then an isolated network is created.
* `is_fenced` (Optional) Fencing allows identical virtual machines in different vApp networks connect to organization VDC networks that are accessed in this vApp. Default is false.
* `retain_ip_mac_enabled` - (Optional) Specifies whether the network resources such as IP/MAC of router will be retained across deployments. Configurable when `is_fenced` is true.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing vApp Org Network can be [imported][docs-import] into this resource
via supplying the full dot separated path for vApp Org Network. An example is below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_vapp_org_network.imported org-name.vdc-name.vapp-name.org-network-name
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

The above command would import the vApp Org Network named `org-network-name` that is defined on vApp 
`vapp-name` which is configured in organization named `my-org` and VDC named `my-org-vdc`.
