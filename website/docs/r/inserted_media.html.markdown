---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_inserted_media"
sidebar_current: "docs-vcd-inserted-media"
description: |-
  Provides a VMware Cloud Director resource for inserting or ejecting media (ISO) file for the VM. Create this resource for inserting the media, and destroy it for ejecting.
---

# vcd\_inserted\_media

Provides a VMware Cloud Director resource for inserting or ejecting media (ISO) file for the VM. Create this resource for inserting the media, and destroy it for ejecting.

Supported in provider *v2.0+*

## Example Usage

```
resource "vcd_inserted_media" "myInsertedMedia" {
  org     = "my-org"
  vdc     = "my-vcd"
  catalog = "my-catalog"
  name    = "my-iso"

  vapp_name = "my-vApp"
  vm_name   = "my-VM"

  eject_force = true
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional; *v2.0+*) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional; *v2.0+*) The name of VDC to use, optional if defined at provider level
* `catalog` - (Required) The name of the catalog where to find media file
* `name` - (Required) Media file name in catalog which will be inserted to VM
* `vapp_name` - (Required) - The name of vApp to find
* `vm_name` - (Required) - The name of VM to be used to insert media file
* `eject_force` - (Optional; *v2.1+*) Allows to pass answer to question in vCD
"The guest operating system has locked the CD-ROM door and is probably using the CD-ROM. 
Disconnect anyway (and override the lock)?" 
when ejecting from a VM which is powered on. True means "Yes" as answer to question. Default is `true`
