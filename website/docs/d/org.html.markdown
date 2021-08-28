---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org"
sidebar_current: "docs-vcd-data-source-org"
description: |-
  Provides an organization data source.
---

# vcd\_org

Provides a VMware Cloud Director Org data source. An organization can be used to manage catalogs, virtual
data centers, and users.

Supported in provider *v2.5+*

## Example Usage

```hcl
data "vcd_org" "my-org" {
  name = "my-org"
}

resource "vcd_org" "my-org-clone" {
  name                 = "my-org-clone"
  full_name            = data.vcd_org.my-org.full_name
  can_publish_catalogs = data.vcd_org.my-org.can_publish_catalogs
  deployed_vm_quota    = data.vcd_org.my-org.deployed_vm_quota
  stored_vm_quota      = data.vcd_org.my-org.stored_vm_quota
  is_enabled           = data.vcd_org.my-org.is_enabled
  delete_force         = "true"
  delete_recursive     = "true"
  vapp_lease {
    maximum_runtime_lease_in_sec          = data.vcd_org.my-org.vapp_lease.0.maximum_runtime_lease_in_sec
    power_off_on_runtime_lease_expiration = data.vcd_org.my-org.vapp_lease.0.power_off_on_runtime_lease_expiration
    maximum_storage_lease_in_sec          = data.vcd_org.my-org.vapp_lease.0.maximum_storage_lease_in_sec
    delete_on_storage_lease_expiration    = data.vcd_org.my-org.vapp_lease.0.delete_on_storage_lease_expiration
  }
  vapp_template_lease {
    maximum_storage_lease_in_sec       = data.vcd_org.my-org.vapp_template_lease.0.maximum_storage_lease_in_sec
    delete_on_storage_lease_expiration = data.vcd_org.my-org.vapp_template_lease.0.delete_on_storage_lease_expiration
  }
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Org name

## Attribute Reference

* `full_name` - Org full name
* `is_enabled` - True if this organization is enabled (allows login and all other operations).
* `description` - Org description.
* `deployed_vm_quota` - Maximum number of virtual machines that can be deployed simultaneously by a member of this organization.
* `stored_vm_quota` - Maximum number of virtual machines in vApps or vApp templates that can be stored in an undeployed state by a member of this organization.
* `can_publish_catalogs` - True if this organization is allowed to share catalogs.
* `delay_after_power_on_seconds` - Specifies this organization's default for virtual machine boot delay after power on.
* `vapp_lease` - (*v2.7+*) - Defines lease parameters for vApps created in this organization. See [vApp Lease](#vapp-lease) below for details. 
* `vapp_template_lease` - (*v2.7+*) - Defines lease parameters for vApp templates created in this organization. See [vApp Template Lease](#vapp-template-lease) below for details.

<a id="vapp-lease"></a>
## vApp Lease

The `vapp_lease` section contains lease parameters for vApps created in the current organization, as defined below:

* `maximum_runtime_lease_in_sec` - How long vApps can run before they are automatically stopped (in seconds)
* `power_off_on_runtime_lease_expiration` - When true, vApps are powered off when the runtime lease expires. When false, vApps are suspended when the runtime lease expires.
* `maximum_storage_lease_in_sec` - How long stopped vApps are available before being automatically cleaned up (in seconds)
* `delete_on_storage_lease_expiration` - If true, storage for a vApp is deleted when the vApp's lease expires. If false, the storage is flagged for deletion, but not deleted.

<a id="vapp-template-lease"></a>
## vApp Template Lease

The `vapp_template_lease` section contains lease parameters for vApp templates created in the current organization, as defined below:

* `maximum_storage_lease_in_sec` - How long vApp templates are available before being automatically cleaned up (in seconds)
* `delete_on_storage_lease_expiration` - If true, storage for a vAppTemplate is deleted when the vAppTemplate lease expires. If false, the storage is flagged for deletion, but not deleted
