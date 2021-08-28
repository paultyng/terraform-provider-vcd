---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org_group"
sidebar_current: "docs-vcd-resource-org-group"
description: |-
  Provides a VMware Cloud Director Organization group. This can be used to create, update, and delete organization groups defined in SAML or LDAP.
---

# vcd\_org\_group

Provides a VMware Cloud Director Organization group. This can be used to create, update, and delete
organization groups defined in `SAML` or `LDAP`.

Supported in provider *v2.9+*

~> **Note:** This operation requires the rights included in the predefined `Organization
Administrator` role or an equivalent set of rights. `SAML` or `LDAP` must be configured as vCD
does not support local groups and will return HTTP error 403 "This operation is denied." if selected
`provider_type` is not configured.

## Example Usage to add SAML group

```hcl
resource "vcd_org_group" "org1" {
  org = "org1"

  provider_type = "SAML"
  name          = "Org1-AdminGroup"
  role          = "Organization Administrator"
}
```

## Example Usage to add LDAP group

```hcl
resource "vcd_org_group" "org1" {
  org = "org1"

  provider_type = "INTEGRATED"
  name          = "ldap-group"
  role          = "Organization Administrator"
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the VDC belongs. Optional if defined at provider level.
* `name` - (Required) A unique name for the group.
* `description` - (Optional) The description of Organization group
* `provider_type` - (Required) Identity provider type for this this group. One of `SAML` or
  `INTEGRATED`. **Note** `LDAP` must be configured to create `INTEGRATED` groups and names must
  match `LDAP` group names. If LDAP is not configured - it will return 403 errors.
* `role` - (Required) The role of the group. Role names can be retrieved from the organization. Both built-in roles and
  custom built can be used. The roles normally available are:
    * `Organization Administrator`
    * `Catalog Author`
    * `vApp Author`
    * `vApp User`
    * `Console Access Only`
    * `Defer to Identity Provider`


## Attribute Reference

The following attributes are exported on this resource:

* `id` - The ID of the Organization group

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing group can be [imported][docs-import] into this resource via supplying the full dot separated path for an
org group. For example, using this structure, representing an existing group that was **not** created using Terraform:

```hcl
resource "vcd_org_group" "my-admin-group" {
  org           = "my-org"
  provider_type = "SAML"
  name          = "my-admin-group"
  role          = "Organization Administrator"
}
```

You can import such group into terraform state using this command

```
terraform import vcd_org_group.my-admin-group my-org.my-admin-group
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/
