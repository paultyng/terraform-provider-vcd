---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_access_control"
sidebar_current: "docs-vcd-resource-vapp-access-control"
description: |-
  Provides a VMware Cloud Director Access Control structure for a vApp.
---

# vcd\_vapp\_access\_control

Provides a VMware Cloud Director Access Control structure for a vApp. This can be used to create, update, and delete access control structures for a vApp.

~> **Warning:** The access control info is tied to a vApp. Thus, there could be only one instance per vApp. Using a different
definition for the same vApp ID will result in a previous instance to be overwritten.

-> **Note:** Access control operations run in tenant context, meaning that, even if the user is a system administrator,
in every request it uses headers items that define the tenant context as restricted to the organization to which the vApp belongs.

Supported in provider *v3.0+*

## Example Usage

```hcl

data "vcd_org_user" "ac-admin1" {
  name = "ac-admin1"
}

data "vcd_org_user" "ac-vapp-creator2" {
  name = "ac-vapp-creator2"
}

data "vcd_vapp" "Vapp-AC-0" {
  name = "Vapp-AC-0"
}

data "vcd_vapp" "Vapp-AC-1" {
  name = "Vapp-AC-1"
}

data "vcd_vapp" "Vapp-AC-2" {
  name = "Vapp-AC-2"
}

resource "vcd_vapp_access_control" "AC-not-shared" {

  vapp_id = data.vcd_vapp.Vapp-AC-0.id

  shared_with_everyone = false
}


resource "vcd_vapp_access_control" "AC-global" {

  vapp_id = data.vcd_vapp.Vapp-AC-1.id

  shared_with_everyone  = true
  everyone_access_level = "Change"
}

resource "vcd_vapp_access_control" "AC-users" {
  vapp_id = data.vcd_vapp.Vapp-AC-2.id

  shared_with_everyone = false

  shared_with {
    user_id      = data.vcd_org_user.ac-admin1.id
    access_level = "FullControl"
  }
  shared_with {
    user_id      = data.vcd_org_user.ac-vapp-creator2.id
    access_level = "Change"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the vApp belongs. Optional if defined at provider level.
* `vdc` - (Optional) The name of organization to which the vApp belongs. Optional if defined at provider level.
* `vapp_id` - (Required) A unique identifier for the vApp.
* `shared_with_everyone` - (Required) Whether the vApp is shared with everyone. If any `shared_with` blocks are included,
  this property cannot be used.
* `everyone_access_level` - (Optional) Access level when the vApp is shared with everyone (one of `ReadOnly`, `Change`, 
`FullControl`). Required if `shared_with_everyone` is set.
* `shared_with` - (Optional) one or more blocks defining a subject to which we are sharing. 
   See [shared_with](#shared_with) below for detail. It cannot be used if `shared_with_everyone` is set.


## shared_with

* `user_id` - (Optional) The ID of a user with which we are sharing. Required if `group_id` is not set.
* `group_id` - (Optional) The ID of a group with which we are sharing. Required if `user_id` is not set.
* `access_level` - (Required) The access level for the user or group to which we are sharing. (One of `ReadOnly`, `Change`, `FullControl`)
* `subject_name` - (Computed) the name of the subject (group or user) with which we are sharing.


## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing `access_control_vapp` can be [imported][docs-import] into this resource via supplying its full dot separated path.
For example, using this structure, representing an existing access control structure that was **not** created using Terraform:

```hcl
resource "vcd_vapp_access_control" "my-ac" {
  org = "my-org"
  vdc = "my-vdc"

  vapp_id = "my-vapp"
}
```

You can import such structure into terraform state using one of these commands

```
terraform import vcd_vapp_access_control.my-ac my-org.my-vdc.vapp-id
terraform import vcd_vapp_access_control.my-ac my-org.my-vdc.vapp-name
```

terraform will import the structure using either the vApp name or its ID.


NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the access control structure as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the access control stored properties.

If you don't know the vApp ID and want to see which ones are available, you can run:

```
terraform import vcd_vapp_access_control.my-ac list@my-org.my-vdc.any-string
```

Terraform will exit with an error message containing the list of available vApps.
