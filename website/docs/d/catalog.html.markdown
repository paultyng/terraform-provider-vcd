---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_catalog"
sidebar_current: "docs-vcd-data-source-catalog"
description: |-
  Provides a catalog data source.
---

# vcd\_catalog

Provides a VMware Cloud Director Catalog data source. A Catalog can be used to manage catalog items and media items.

Supported in provider *v2.5+*

## Example Usage

```hcl
data "vcd_catalog" "my-cat" {
  org  = "my-org"
  name = "my-cat"
}

resource "vcd_catalog_item" "myItem" {
  org     = data.vcd_catalog.my-cat.org
  catalog = data.vcd_catalog.my-cat.name

  name                 = "myItem"
  description          = "Belongs to ${data.vcd_catalog.my-cat.id}"
  ova_path             = "/path/to/test_vapp_template.ova"
  upload_piece_size    = 5
  show_upload_progress = "true"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional, but required if not set at provider level) Org name 
* `name` - (Required) Catalog name (optional when `filter` is used)
* `filter` - (Optional; *2.9+*) Retrieves the data source using one or more filter parameters

## Attribute Reference

* `description` - Catalog description.

## Filter arguments

(Supported in provider *v2.9+*)

* `name_regex` (Optional) matches the name using a regular expression.
* `date` (Optional) is an expression starting with an operator (`>`, `<`, `>=`, `<=`, `==`), followed by a date, with
  optional spaces in between. For example: `> 2020-02-01 12:35:00.523Z`
  The filter recognizes several formats, but one of `yyyy-mm-dd [hh:mm[:ss[.nnnZ]]]` or `dd-MMM-yyyy [hh:mm[:ss[.nnnZ]]]`
  is recommended.
  Comparison with equality operator (`==`) need to define the date to the microseconds.
* `latest` (Optional) If `true`, retrieve the latest item among the ones matching other parameters. If no other parameters
  are set, it retrieves the newest item.
* `earliest` (Optional) If `true`, retrieve the earliest item among the ones matching other parameters. If no other parameters
  are set, it retrieves the oldest item.
* `metadata` (Optional) One or more parameters that will match metadata contents.

See [Filters reference](/docs/providers/vcd/guides/data_source_filters.html) for details and examples.

