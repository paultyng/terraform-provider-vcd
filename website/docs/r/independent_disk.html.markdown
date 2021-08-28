---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_independent_disk"
sidebar_current: "docs-vcd-independent-disk"
description: |-
  Provides a VMware Cloud Director independent disk resource. This can be used to create and delete independent disks.
---

# vcd\_independent\_disk

Provides a VMware Cloud Director independent disk resource. This can be used to create and delete independent disks.

## Example Usage

```hcl
resource "vcd_independent_disk" "myNewIndependentDisk" {  
  vdc             = "my-vcd"
  
  name            = "logDisk"
  size_in_mb      = "1024"
  bus_type        = "SCSI"
  bus_sub_type    = "VirtualSCSI"
  storage_profile = "external"
}

resource "vcd_vapp_vm" "web2" {
  vapp_name     = "${vcd_vapp.web.name}"

...
  
  disk {
    name = "${vcd_independent_disk.myNewIndependentDisk.name}"
    bus_number = 1
    unit_number = 0
  }

  depends_on = ["vcd_independent_disk.myNewIndependentDisk"]
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `name` - (Required) Disk name
* `size_in_mb` - (Required, *v3.0+*) Size of disk in MB.
* `bus_type` - (Optional) Disk bus type. Values can be: `IDE`, `SCSI`, `SATA` 
* `bus_sub_type` - (Optional) Disk bus subtype. Values can be: `buslogic`, `lsilogic`, `lsilogicsas`, `VirtualSCSI` for `SCSI` and `ahci` for `SATA`
* `storage_profile` - (Optional) The name of storage profile where disk will be created

## Attribute reference

Supported in provider *v2.5+*

* `iops` - (Computed) IOPS request for the created disk
* `owner_name` - (Computed) The owner name of the disk
* `datastore_name` - (Computed) Data store name. Readable only for system user.
* `is_attached` - (Computed) True if the disk is already attached

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing independent disk can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of org-name.vdc-name.disk-id
For example, using this structure, representing a independent disk that was **not** created using Terraform:

```hcl
resource "vcd_independent_disk" "tf-myDisk" {
  vdc  = "my-vdc"
  name = "my-disk"
}
```

You can import such independent disk into terraform state using this command

```
terraform import vcd_independent_disk.tf-myDisk org-name.vdc-name.my-disk-id
```

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.

### Listing independent disk IDs

If you want to list IDs there is a special command **`terraform import vcd_independent_disk.imported list@org-name.vdc-name.my-independent-disk-name`**
where `org-name` is the organization used, `vdc-name` is vDC name and `my-independent-disk-name`
is independent disk name. The output for this command should look similar to the one below:

```shell
$ terraform import vcd_independent_disk.imported list@org-name.vdc-name.my-independent-disk-name
vcd_independent_disk.Disk_import: Importing from ID "list@org-name.vdc-name.my-independent-disk-name"...
Retrieving all disks by name
No  ID                                                      Name    Description Size
--  --                                                      ----    ------      ----
1  urn:vcloud:disk:1bbc273d-7701-4f06-97be-428b46b0805e     diskV2  loging      78946548
2  urn:vcloud:disk:6e1c996f-48b8-4e78-8111-a6407188d8b6     diskV2              5557452

Error: resource was not imported! resource id must be specified in one of these formats:
'org-name.vdc-name.my-independent-disk-id' to import by rule id
'list@org-name.vdc-name.my-independent-disk-name' to get a list of disks with their IDs
```

Now to import disk with ID urn:vcloud:disk:1bbc273d-7701-4f06-97be-428b46b0805e one could supply this command:

```shell
$ terraform import vcd_independent_disk.imported list@org-name.vdc-name.urn:vcloud:disk:1bbc273d-7701-4f06-97be-428b46b0805e
```
