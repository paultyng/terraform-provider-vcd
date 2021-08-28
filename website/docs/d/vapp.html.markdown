---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp"
sidebar_current: "docs-vcd-datasource-vapp"
description: |-
  Provides a VMware Cloud Director vApp data source. This can be used to reference vApps.
---

# vcd\_vapp

Provides a VMware Cloud Director vApp data source. This can be used to reference vApps.

Supported in provider *v2.5+*

## Example Usage


```hcl
data "vcd_vapp" "test-tf" {
  name = "test-tf"
  org  = "tf"
  vdc  = "vdc-tf"
}

output "id" {
  value = data.vcd_vapp.test-tf.id
}

output "name" {
  value = data.vcd_vapp.test-tf.name
}

output "description" {
  value = data.vcd_vapp.test-tf.description
}

output "href" {
  value = data.vcd_vapp.test-tf.href
}

output "status_text" {
  value = data.vcd_vapp.test-tf.status_text
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the vApp
* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level

## Attribute reference

* `description` An optional description for the vApp
* `href` - The vApp Hyper Reference
* `metadata` -  Key value map of metadata to assign to this vApp. Key and value can be any string. 
* `guest_properties` -  Key value map of vApp guest properties.
* `status` -  The vApp status as a numeric code
* `status_text` -  The vApp status as text.
