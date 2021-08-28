---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org"
sidebar_current: "docs-vcd-resource-org"
description: |-
  Provides a VMware Cloud Director Organization resource. This can be used to create  delete, and update an organization.
---

# vcd\_org

Provides a VMware Cloud Director Org resource. This can be used to create, update, and delete an organization.
Requires system administrator privileges.

Supported in provider *v2.0+*

## Example Usage

```hcl
provider "vcd" {
  user     = "${var.admin_user}"
  password = "${var.admin_password}"
  org      = "System"
  url      = "https://AcmeVcd/api"
}

resource "vcd_org" "my-org" {
  name             = "my-org"
  full_name        = "My organization"
  description      = "The pride of my work"
  is_enabled       = "true"
  delete_recursive = "true"
  delete_force     = "true"

  vapp_lease {
    maximum_runtime_lease_in_sec          = 3600 # 1 hour
    power_off_on_runtime_lease_expiration = true
    maximum_storage_lease_in_sec          = 0 # never expires
    delete_on_storage_lease_expiration    = false
  }
  vapp_template_lease {
    maximum_storage_lease_in_sec       = 604800 # 1 week
    delete_on_storage_lease_expiration = true
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Org name
* `full_name` - (Required) Org full name
* `delete_recursive` - (Required) - pass `delete_recursive`=true as query parameter to remove an organization or VDC and any objects it contains that are in a state that normally allows removal.
* `delete_force` - (Required) - pass `delete_force=true` and `delete_recursive=true` to remove an organization or VDC and any objects it contains, regardless of their state.
* `is_enabled` - (Optional) - True if this organization is enabled (allows login and all other operations). Default is `true`.
* `description` - (Optional) - Org description. Default is empty.
* `deployed_vm_quota` - (Optional) - Maximum number of virtual machines that can be deployed simultaneously by a member of this organization. Default is unlimited (0)
* `stored_vm_quota` - (Optional) - Maximum number of virtual machines in vApps or vApp templates that can be stored in an undeployed state by a member of this organization. Default is unlimited (0)
* `can_publish_catalogs` - (Optional) - True if this organization is allowed to share catalogs. Default is `true`.
* `delay_after_power_on_seconds` - (Optional) - Specifies this organization's default for virtual machine boot delay after power on. Default is `0`.
* `vapp_lease` - (Optional; *v2.7+*) - Defines lease parameters for vApps created in this organization. See [vApp Lease](#vapp-lease) below for details. 
* `vapp_template_lease` - (Optional; *v2.7+*) - Defines lease parameters for vApp templates created in this organization. See [vApp Template Lease](#vapp-template-lease) below for details.

<a id="vapp-lease"></a>
## vApp Lease

The `vapp_lease` section contains lease parameters for vApps created in the current organization, as defined below:

* `maximum_runtime_lease_in_sec` - (Required) - How long vApps can run before they are automatically stopped (in seconds). 0 means never expires. Values accepted from 3600+
<br>Note: Default when the whole `vapp_lease` block is omitted is 604800 (7 days) but may vary depending on vCD version
* `power_off_on_runtime_lease_expiration` - (Required) - When true, vApps are powered off when the runtime lease expires. When false, vApps are suspended when the runtime lease expires.
<br>Note: Default when the whole `vapp_lease` block is omitted is false
* `maximum_storage_lease_in_sec` - (Required) - How long stopped vApps are available before being automatically cleaned up (in seconds). 0 means never expires. Regular values accepted from 3600+
<br>Note: Default when the whole `vapp_lease` block is omitted is 2592000 (30 days) but may vary depending on vCD version
* `delete_on_storage_lease_expiration` - (Required) - If true, storage for a vApp is deleted when the vApp's lease expires. If false, the storage is flagged for deletion, but not deleted.
<br>Note: Default when the whole `vapp_lease` block is omitted is false

<a id="vapp-template-lease"></a>
## vApp Template Lease

The `vapp_template_lease` section contains lease parameters for vApp templates created in the current organization, as defined below:

* `maximum_storage_lease_in_sec` - (Required) - How long vApp templates are available before being automatically cleaned up (in seconds). 0 means never expires. Regular values accepted from 3600+
<br>Note: Default when the whole `vapp_template_lease` block is omitted is 2592000 (30 days) but may vary depending on vCD version
* `delete_on_storage_lease_expiration` - (Required) - If true, storage for a vAppTemplate is deleted when the vAppTemplate lease expires. If false, the storage is flagged for deletion, but not deleted. 
<br>Note: Default when the whole `vapp_template_lease` block is omitted is false

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing Org can be [imported][docs-import] into this resource via supplying the path for an Org. Since the Org is
at the top of the vCD hierarchy, the path corresponds to the Org name.
For example, using this structure, representing an existing Org that was **not** created using Terraform:

```hcl
resource "vcd_org" "my-orgadmin" {
  name             = "my-org"
  full_name        = "guessing"
  delete_recursive = "true"
  delete_force     = "true"
}
```

You can import such organization into terraform state using this command

```
terraform import vcd_org.my-org my-org
```

[docs-import]:https://www.terraform.io/docs/import/

The state (in `terraform.tfstate`) would look like this:

```json
{
  "version": 4,
  "terraform_version": "0.12.0",
  "serial": 1,
  "lineage": "4f328a1d-3ac3-a1be-b739-c1edde689335",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "vcd_org",
      "name": "my-org",
      "provider": "provider.vcd",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "can_publish_catalogs": true,
            "delay_after_power_on_seconds": null,
            "delete_force": null,
            "delete_recursive": null,
            "deployed_vm_quota": 50,
            "description": "",
            "full_name": "my-org",
            "id": "urn:vcloud:org:875e81c4-3d7a-4bf4-b7db-9d0abe0f0b0d",
            "is_enabled": true,
            "name": "my-org",
            "stored_vm_quota": 50
          }
        }
      ]
    }
  ]
}
```
After that, you can expand the configuration file and either update or delete the org as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the Org's stored properties.

## Sources

* [OrgType](https://code.vmware.com/apis/287/vcloud#/doc/doc/types/OrgType.html)
* [ReferenceType](https://code.vmware.com/apis/287/vcloud#/doc/doc/types/ReferenceType.html)
* [Org deletion](https://code.vmware.com/apis/287/vcloud#/doc/doc/operations/DELETE-Organization.html)

