---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_catalog_media"
sidebar_current: "docs-vcd-resource-catalog-media"
description: |-
  Provides a VMware Cloud Director media resource. This can be used to upload and delete media (ISO) file inside a catalog.
---

# vcd\_catalog\_media

Provides a VMware Cloud Director media resource. This can be used to upload media to catalog and delete it.

Supported in provider *v2.0+*

## Example Usage

```hcl
resource "vcd_catalog_media" "myNewMedia" {
  org     = "my-org"
  catalog = "my-catalog"

  name                 = "my iso"
  description          = "new os versions"
  media_path           = "/home/user/file.iso"
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
* `catalog` - (Required) The name of the catalog where to upload media file
* `name` - (Required) Media file name in catalog
* `description` - (Optional) - Description of media file
* `media_path` - (Required) - Absolute or relative path to file to upload
* `upload_piece_size` - (Optional) - size in MB for splitting upload size. It can possibly impact upload performance. Default 1MB.
* `show_upload_progress` - (Optional) - Default false. Allows to see upload progress
* `metadata` - (Optional; *v2.5+*) Key value map of metadata to assign

## Attribute reference

Supported in provider *v2.5+*

* `is_iso` - (Computed) returns True if this media file is ISO
* `owner_name` - (Computed) returns owner name
* `is_published` - (Computed) returns True if this media file is in a published catalog
* `creation_date` - (Computed) returns creation date
* `size` - (Computed) returns media storage in Bytes
* `status` - (Computed) returns media status
* `storage_profile_name` - (Computed) returns storage profile name

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing media item can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of org-name.catalog-name.media-name
For example, using this structure, representing a media item that was **not** created using Terraform:

```hcl
resource "vcd_catalog_media" "tf-mymedia" {
  org     = "my-org"
  catalog = "my-catalog"
  name    = "my-media"
}
```

You can import such catalog media into terraform state using this command

```
terraform import vcd_catalog_media.tf-mymedia my-org.my-catalog.my-media
```

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.
