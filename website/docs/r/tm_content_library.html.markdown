---
layout: "vcd"
page_title: "VMware Cloud Foundation Tenant Manager: vcd_tm_content_library"
sidebar_current: "docs-vcd-resource-tm-content-library"
description: |-
  Provides a VMware Cloud Foundation Tenant Manager Content Library resource. This can be used to manage Content Libraries.
---

# vcd\_tm\_content\_library

Provides a VMware Cloud Foundation Tenant Manager Content Library resource. This can be used to manage Content Libraries.

This resource is exclusive to **VMware Cloud Foundation Tenant Manager**. Supported in provider *v4.0+*

## Example Usage for a Provider Content Library

```hcl
data "vcd_tm_region" "region" {
  name = "My Region"
}

data "vcd_tm_storage_class" "sc" {
  region_id = data.vcd_tm_region.region.id
  name      = "vSAN Default Storage Policy"
}

resource "vcd_tm_content_library" "cl" {
  name        = "My Library"
  description = "A simple library"
  storage_class_ids = [
    data.vcd_tm_storage_class.sc.id
  ]
}
```

## Example Usage for a Tenant Content Library

// TODO: TM (Tenant support is not implemented yet)

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Content Library
* `storage_class_ids` - (Required) A set of [Storage Class IDs](/providers/vmware/vcd/latest/docs/data-sources/tm_storage_class) used by this Content Library
* `auto_attach` - (Optional) Defaults to `true`. For Tenant Content Libraries this field represents whether this Content Library should be
  automatically attached to all current and future namespaces in the tenant organization. If a value of `false` is supplied, then this
  Tenant Content Library will only be attached to namespaces that explicitly request it. For Provider Content Libraries this field is not needed
  for creation and will always be returned as true. This field cannot be updated after Content Library creation
* `description` - (Optional) The description of the Content Library
* `subscription_config` - (Optional) A block representing subscription settings of a Content Library:
  *  `subscription_url` - Subscription url of this Content Library
  *  `password` - Password to use to authenticate with the publisher
  *  `need_local_copy` - Whether to eagerly download content from publisher and store it locally

## Attribute Reference

* `creation_date` - The ISO-8601 timestamp representing when this Content Library was created
* `is_shared` - Whether this Content Library is shared with other Organziations
* `is_subscribed` - Whether this Content Library is subscribed from an external published library
* `library_type` - The type of content library, can be either `PROVIDER` (Content Library that is scoped to a provider) or 
  `TENANT` (Content Library that is scoped to a tenant organization)
* `owner_org_id` - The reference to the Organization that the Content Library belongs to
* `version_number` - Version number of this Content library 

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. However, an experimental feature in Terraform 1.5+ allows also code generation.
See [Importing resources][importing-resources] for more information.

An existing Content Library can be [imported][docs-import] into this resource via supplying its name.
For example, using this structure, representing an existing Content Library that was **not** created using Terraform:

```hcl
resource "vcd_tm_content_library" "cl" {
  name = "My Already Existing Library"
}
```

You can import such Content Library into terraform state using this command

```
terraform import vcd_tm_content_library.cl "My Already Existing Library"
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

After that, you can expand the configuration file and either update or delete the Content Library as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the Content Library's stored properties.

[importing-resources]:https://registry.terraform.io/providers/vmware/vcd/latest/docs/guides/importing_resources