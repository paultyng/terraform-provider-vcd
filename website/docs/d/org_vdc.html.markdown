---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org_vdc"
sidebar_current: "docs-vcd-data-source-org-vdc"
description: |-
  Provides an organization VDC data source.
---

# vcd\_org\_vdc

Provides a VMware Cloud Director Organization VDC data source. An Organization VDC can be used to reference a VCD and use its 
data within other resources or data sources.

-> **Note:** This resource supports NSX-T and NSX-V based Org VDCs

Supported in provider *v2.5+*

## Example Usage

```hcl
data "vcd_org_vdc" "my-org-vdc" {
  org  = "my-org"
  name = "my-vdc"
}

output "provider_vdc" {
  value = data.vcd_org_vdc.my-org-vdc.provider_vdc_name
}

```

## Argument Reference

The following arguments are supported:

* `org` - (Optional, but required if not set at provider level) Org name 
* `name` - (Required) Organization VDC name

## Attribute reference

All attributes defined in [organization VDC resource](/docs/providers/vcd/r/org_vdc.html#attribute-reference) are supported.

