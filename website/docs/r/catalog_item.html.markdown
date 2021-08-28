---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_catalog_item"
sidebar_current: "docs-vcd-resource-catalog-item"
description: |-
  Provides a VMware Cloud Director catalog item resource. This can be used to upload and delete OVA file inside a catalog.
---

# vcd\_catalog\_item

Provides a VMware Cloud Director catalog item resource. This can be used to upload OVA to catalog and delete it.

Supported in provider *v2.0+*

## Example Usage

```hcl
resource "vcd_catalog_item" "myNewCatalogItem" {
  org     = "my-org"
  catalog = "my-catalog"

  name                 = "my ova"
  description          = "new vapp template"
  ova_path             = "/home/user/file.ova"
  upload_piece_size    = 10
  show_upload_progress = true

  metadata = {
    license = "public"
    version = "v1"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `catalog` - (Required) The name of the catalog where to upload OVA file
* `name` - (Required) Item name in catalog
* `description` - (Optional) - Description of item
* `ova_path` - (Required) - Absolute or relative path to file to upload
* `upload_piece_size` - (Optional) - Size in MB for splitting upload size. It can possibly impact upload performance. Default 1MB.
* `show_upload_progress` - (Optional) - Default false. Allows to see upload progress
* `metadata` - (Optional; *v2.5+*) Key value map of metadata to assign

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing catalog item can be [imported][docs-import] into this resource via supplying the full dot separated path for a
catalog item. For example, using this structure, representing an existing catalog item that was **not** created using Terraform:

```hcl
resource "vcd_catalog_item" "my-item" {
  org      = "my-org"
  catalog  = "my-catalog"
  name     = "my-item"
  ova_path = "guess"
}
```

You can import such catalog item into terraform state using this command

```
terraform import vcd_catalog_item.my-item my-org.my-catalog.my-item
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the catalog item as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the item's stored properties.
