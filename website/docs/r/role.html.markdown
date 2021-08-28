---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_role"
sidebar_current: "docs-vcd-resource-role"
description: |-
 Provides a VMware Cloud Director role. This can be used to create, modify, and delete roles.
---

# vcd\_role

Provides a VMware Cloud Director role. This can be used to create, modify, and delete roles.

Supported in provider *v3.3+*

## Example Usage

```hcl
resource "vcd_role" "new-role" {
  org         = "my-org"
  name        = "new-role"
  description = "new role from CLI"
  rights = [
    "Catalog: Add vApp from My Cloud",
    "Catalog: Edit Properties",
    "Catalog: View Private and Shared Catalogs",
    "Organization vDC Compute Policy: View",
    "vApp Template / Media: Edit",
    "vApp Template / Media: View",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `name` - (Required) The name of the role.
* `description` - (Required) A description of the role
* `rights` - (Optional) List of rights assigned to this role

## Attribute Reference

* `read_only` - Whether this role is read-only
* `bundle_key` - Key used for internationalization

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing role can be [imported][docs-import] into this resource via supplying the full dot separated path for a role.
For example, using this structure, representing an existing role that was **not** created using Terraform:

```hcl

resource "vcd_role" "catalog-author" {
  org  = "my-org"
  name = "Catalog Author"
}
```

You can import such role into terraform state using this command

```
terraform import vcd_role.catalog-author "my-org.Catalog Author"
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the role as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the role's stored properties.

## More information

See [Roles management](/docs/providers/vcd/guides/roles_management.html) for a broader description of how roles and
rights work together.
