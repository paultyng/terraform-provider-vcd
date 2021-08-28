---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_role"
sidebar_current: "docs-vcd-data-source-role"
description: |-
 Provides a VMware Cloud Director role. This can be used to read roles.
---

# vcd\_role

Provides a VMware Cloud Director role data source. This can be used to read roles.

Supported in provider *v3.3+*

## Example Usage

```hcl
data "vcd_role" "vapp-author" {
  org  = "my-org"
  name = "vApp Author"
}
```
```
Sample output:

vapp-author = {
  "bundle_key" = "ROLE_VAPP_AUTHOR"
  "description" = "Rights given to a user who uses catalogs and creates vApps"
  "id" = "urn:vcloud:role:53256466-221f-3f1f-8cea-2fcfc7ab9ef7"
  "name" = "vApp Author"
  "org" = "datacloud"
  "read_only" = true
  "rights" = toset([
    "Catalog: Add vApp from My Cloud",
    "Catalog: View Private and Shared Catalogs",
    "Organization vDC Compute Policy: View",
    "Organization vDC Named Disk: Create",
    "Organization vDC Named Disk: Delete",
    "Organization vDC Named Disk: Edit Properties",
    "Organization vDC Named Disk: View Properties",
    "Organization vDC Network: View Properties",
    "Organization vDC: VM-VM Affinity Edit",
    "Organization: View",
    "UI Plugins: View",
    "VAPP_VM_METADATA_TO_VCENTER",
    "vApp Template / Media: Copy",
    "vApp Template / Media: Edit",
    "vApp Template / Media: View",
    "vApp Template: Checkout",
    "vApp: Copy",
    "vApp: Create / Reconfigure",
    "vApp: Delete",
    "vApp: Download",
    "vApp: Edit Properties",
    "vApp: Edit VM CPU",
    "vApp: Edit VM Hard Disk",
    "vApp: Edit VM Memory",
    "vApp: Edit VM Network",
    "vApp: Edit VM Properties",
    "vApp: Manage VM Password Settings",
    "vApp: Power Operations",
    "vApp: Sharing",
    "vApp: Snapshot Operations",
    "vApp: Upload",
    "vApp: Use Console",
    "vApp: VM Boot Options",
    "vApp: View ACL",
    "vApp: View VM metrics",
  ])
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `name` - (Required) The name of the role.

## Attribute Reference

* `read_only` - Whether this role is read-only
* `description` - A description of the role
* `bundle_key` - Key used for internationalization.
* `rights` - List of rights assigned to this role

## More information

See [Roles management](/docs/providers/vcd/guides/roles_management.html) for a broader description of how roles and
rights work together.
