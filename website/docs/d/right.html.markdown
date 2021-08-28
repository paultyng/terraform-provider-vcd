---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_right"
sidebar_current: "docs-vcd-data-source-right"
description: |-
 Provides a VMware Cloud Director Organization Right data source. This can be used to read existing rights
---

# vcd\_right

Provides a data source for available rights.

Supported in provider *v3.3+*

## Example usage

```hcl
data "vcd_right" "some-right" {
  name = "Catalog: Add vApp from My Cloud"
}

output "some-right" {
  value = data.vcd_right.some-right
}
```

```
Sample output:

some-right = {
  "bundle_key" = "RIGHT_CATALOG_ADD_VAPP_FROM_MY_CLOUD"
  "category_id" = "urn:vcloud:rightsCategory:c32516ba-bc5b-3c47-ab8c-e1bfc223253c"
  "description" = "Add a vApp from My Cloud"
  "id" = "urn:vcloud:right:4886663f-ae31-37fc-9a70-3dbe2f24a8c5"
  "implied_rights" = toset([
    {
      "id" = "urn:vcloud:right:1aa46727-6192-365d-b571-5ce51beb3b48"
      "name" = "vApp Template / Media: View"
    },
    {
      "id" = "urn:vcloud:right:3eedbfb4-c4a3-373d-b4b5-d76ca363ab50"
      "name" = "vApp Template / Media: Edit"
    },
    {
      "id" = "urn:vcloud:right:fa4ce8f8-c640-3b65-8fa5-a863b56c3d51"
      "name" = "Catalog: View Private and Shared Catalogs"
    },
  ])
  "name" = "Catalog: Add vApp from My Cloud"
  "right_type" = "MODIFY"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the right.

## Attribute reference

* `description` - A description of the right
* `category_id` - The ID of the category for this right
* `bundle_key` - Key used for internationalization
* `right type` - Type of the right (VIEW or MODIFY)
* `implied_rights` - List of rights that are implied with this one

## More information

See [Roles management](/docs/providers/vcd/guides/roles_management.html) for a broader description of how roles and
rights work together.
